package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v3"
)

type ToolDefinition struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	InputSchema openai.FunctionParameters `json:"input_schema"`
	Function    func(input json.RawMessage, log interface{ Debug(format string, args ...interface{}); Error(format string, args ...interface{}); Warn(format string, args ...interface{}) }) (string, error)
}

func (t ToolDefinition) FunctionDefinition() openai.FunctionDefinitionParam {
	def := openai.FunctionDefinitionParam{
		Name:       t.Name,
		Parameters: t.InputSchema,
	}
	if t.Description != "" {
		def.Description = openai.String(t.Description)
	}
	return def
}

func (t ToolDefinition) ToolConfig() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(t.FunctionDefinition())
}

// Input structs
type ReadFileInput struct {
	Path string `json:"path" jsonschema_description:"The relative path of a file in the working directory."`
}

type ListFilesInput struct {
	Path string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
}

type BashInput struct {
	Command string `json:"command" jsonschema_description:"The bash command to execute."`
}

type EditFileInput struct {
	Path   string `json:"path" jsonschema_description:"The path to the file"`
	OldStr string `json:"old_str" jsonschema_description:"Text to search for - must match exactly and must only have one match exactly"`
	NewStr string `json:"new_str" jsonschema_description:"Text to replace old_str with"`
}

type CodeSearchInput struct {
	Pattern       string `json:"pattern" jsonschema_description:"The search pattern or regex to look for"`
	Path          string `json:"path,omitempty" jsonschema_description:"Optional path to search in (file or directory)"`
	FileType      string `json:"file_type,omitempty" jsonschema_description:"Optional file extension to limit search to (e.g., 'go', 'js', 'py')"`
	CaseSensitive bool   `json:"case_sensitive,omitempty" jsonschema_description:"Whether the search should be case sensitive (default: false)"`
}

// Schemas
var ReadFileInputSchema = GenerateSchema[ReadFileInput]()
var ListFilesInputSchema = GenerateSchema[ListFilesInput]()
var BashInputSchema = GenerateSchema[BashInput]()
var EditFileInputSchema = GenerateSchema[EditFileInput]()
var CodeSearchInputSchema = GenerateSchema[CodeSearchInput]()

// Tool definitions
var ReadFileDefinition = ToolDefinition{
	Name:        "read_file",
	Description: "Read the contents of a given relative file path. Use this when you want to see what's inside a file. Do not use this with directory names.",
	InputSchema: ReadFileInputSchema,
	Function:    ReadFile,
}

var ListFilesDefinition = ToolDefinition{
	Name:        "list_files",
	Description: "List files and directories at a given path. If no path is provided, lists files in the current directory.",
	InputSchema: ListFilesInputSchema,
	Function:    ListFiles,
}

var BashDefinition = ToolDefinition{
	Name:        "bash",
	Description: "Execute a bash command and return its output. Use this to run shell commands.",
	InputSchema: BashInputSchema,
	Function:    Bash,
}

var EditFileDefinition = ToolDefinition{
	Name: "edit_file",
	Description: `Make edits to a text file.
	Replace 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.
	If the file specified with path doesn't exist, it will be created.
	`,
	InputSchema: EditFileInputSchema,
	Function:    EditFile,
}

var CodeSearchDefinition = ToolDefinition{
	Name: "code_search",
	Description: `Search for code patterns using ripgrep (rg).
	Use this to find code patterns, function definitions, variable usage, or any text in the codebase.
	You can search by pattern, file type, or directory.`,
	InputSchema: CodeSearchInputSchema,
	Function:    CodeSearch,
}

// Tool implementations
func ReadFile(input json.RawMessage, log interface{ Debug(format string, args ...interface{}); Error(format string, args ...interface{}); Warn(format string, args ...interface{}) }) (string, error) {
	readFileInput := ReadFileInput{}
	err := json.Unmarshal(input, &readFileInput)
	if err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	log.Debug("Reading file: %s", readFileInput.Path)
	content, err := os.ReadFile(readFileInput.Path)
	if err != nil {
		log.Error("Failed to read file %s: %v", readFileInput.Path, err)
		return "", err
	}
	log.Debug("Successfully read file %s (%d bytes)", readFileInput.Path, len(content))
	return string(content), nil
}

func ListFiles(input json.RawMessage, log interface{ Debug(format string, args ...interface{}); Error(format string, args ...interface{}); Warn(format string, args ...interface{}) }) (string, error) {
	listFilesInput := ListFilesInput{}
	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		return "", fmt.Errorf("invalid input: %w", err)
	}

	dir := "."
	if listFilesInput.Path != "" {
		dir = listFilesInput.Path
	}

	log.Debug("Listing files in directory: %s", dir)

	var files []string
	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dir, path)
		if err != nil {
			return err
		}

		if relPath != "." {
			relPath = filepath.ToSlash(relPath)
			if info.IsDir() {
				files = append(files, relPath+"/")
			} else {
				files = append(files, relPath)
			}
		}

		return nil
	})
	if err != nil {
		log.Error("Failed to list files in %s: %v", dir, err)
		return "", err
	}

	log.Debug("Successfully listed %d items in %s", len(files), dir)

	result, err := json.Marshal(files)
	if err != nil {
		return "", err
	}

	return string(result), nil
}

func Bash(input json.RawMessage, log interface{ Debug(format string, args ...interface{}); Error(format string, args ...interface{}); Warn(format string, args ...interface{}) }) (string, error) {
	bashInput := BashInput{}
	err := json.Unmarshal(input, &bashInput)
	if err != nil {
		return "", err
	}

	log.Debug("Executing bash command: %s", bashInput.Command)
	cmd := exec.Command("nu", "-c", bashInput.Command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Warn("Bash command failed: %s, error: %v", bashInput.Command, err)
		return fmt.Sprintf("Command failed with error: %s\nOutput: %s", err.Error(), string(output)), nil
	}

	log.Debug("Bash command succeeded: %s (output: %d bytes)", bashInput.Command, len(output))
	return strings.TrimSpace(string(output)), nil
}

func EditFile(input json.RawMessage, log interface{ Debug(format string, args ...interface{}); Error(format string, args ...interface{}); Warn(format string, args ...interface{}) }) (string, error) {
	editFileInput := EditFileInput{}
	err := json.Unmarshal(input, &editFileInput)
	if err != nil {
		return "", err
	}

	if editFileInput.Path == "" || editFileInput.OldStr == editFileInput.NewStr {
		log.Error("EditFile failed: invalid input parameters")
		return "", fmt.Errorf("invalid input parameters")
	}

	log.Debug("Editing file: %s (replacing %d chars with %d chars)", editFileInput.Path, len(editFileInput.OldStr), len(editFileInput.NewStr))
	content, err := os.ReadFile(editFileInput.Path)
	if err != nil {
		if os.IsNotExist(err) && editFileInput.OldStr == "" {
			log.Debug("File does not exist, creating new file: %s", editFileInput.Path)
			return createNewFile(editFileInput.Path, editFileInput.NewStr, log)
		}
		log.Error("Failed to read file %s: %v", editFileInput.Path, err)
	}
	oldContent := string(content)

	// Special case: if old_str is empty, we're appending to the file
	var newContent string
	if editFileInput.OldStr == "" {
		newContent = oldContent + editFileInput.NewStr
	} else {
		// count occurrences first to ensure we have exactly one match
		count := strings.Count(oldContent, editFileInput.OldStr)
		if count == 0 {
			log.Error("EditFile failed: old_str not found in file %s", editFileInput.Path)
			return "", fmt.Errorf("old_str not found in file")
		}
		if count > 1 {
			log.Error("EditFile failed: old_str found %d times in file %s, must be unique", count, editFileInput.Path)
			return "", fmt.Errorf("old_str found %d times in file, must be unique", count)
		}

		newContent = strings.Replace(oldContent, editFileInput.OldStr, editFileInput.NewStr, 1)
	}

	err = os.WriteFile(editFileInput.Path, []byte(newContent), 0644)
	if err != nil {
		log.Error("Failed to write file %s: %v", editFileInput.Path, err)
		return "", err
	}

	log.Debug("Successfully edited file %s", editFileInput.Path)
	return "OK", nil
}

func CodeSearch(input json.RawMessage, log interface{ Debug(format string, args ...interface{}); Error(format string, args ...interface{}); Warn(format string, args ...interface{}) }) (string, error) {
	codeSearchInput := CodeSearchInput{}
	err := json.Unmarshal(input, &codeSearchInput)
	if err != nil {
		return "", err
	}

	if codeSearchInput.Pattern == "" {
		log.Error("CodeSearch failed: pattern is required")
		return "", fmt.Errorf("pattern is required")
	}

	log.Debug("Searching for pattern: %s", codeSearchInput.Pattern)

	// Build ripgrep command
	args := []string{"rg", "--line-number", "--with-filename", "--color=never"}

	// Add case sensitivity flag
	if !codeSearchInput.CaseSensitive {
		args = append(args, "--ignore-case")
	}

	// Add file type filter if specified
	if codeSearchInput.FileType != "" {
		args = append(args, "--type", codeSearchInput.FileType)
	}

	// Add pattern
	args = append(args, codeSearchInput.Pattern)

	// Add path if specified
	if codeSearchInput.Path != "" {
		args = append(args, codeSearchInput.Path)
	} else {
		args = append(args, ".")
	}

	log.Debug("Executing ripgrep with args: %v", args)

	cmd := exec.Command(args[0], args[1:]...)
	output, err := cmd.Output()

	// ripgrep returns exit code 1 when no matches are found, which is not an error
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() == 1 {
			log.Debug("No matches found for pattern: %s", codeSearchInput.Pattern)
			return "No matches found", nil
		}
		log.Error("Ripgrep command failed: %v", err)
		return "", fmt.Errorf("search failed: %w", err)
	}

	result := strings.TrimSpace(string(output))
	lines := strings.Split(result, "\n")

	log.Debug("Found %d matches for pattern: %s", len(lines), codeSearchInput.Pattern)

	// Limit output to prevent overwhelming responses
	if len(lines) > 50 {
		result = strings.Join(lines[:50], "\n") + fmt.Sprintf("\n... (showing first 50 of %d matches)", len(lines))
	}

	return result, nil
}

func createNewFile(filePath, content string, log interface{ Debug(format string, args ...interface{}); Error(format string, args ...interface{}); Warn(format string, args ...interface{}) }) (string, error) {
	log.Debug("Creating new file: %s (%d bytes)", filePath, len(content))
	dir := path.Dir(filePath)
	if dir != "." {
		log.Debug("Creating directory: %s", dir)
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Error("Failed to create directory %s: %v", dir, err)
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		log.Error("Failed to write file %s: %v", filePath, err)
		return "", fmt.Errorf("failed to write file: %w", err)
	}

	log.Debug("Successfully created file %s", filePath)
	return fmt.Sprintf("Successfully created file %s", filePath), nil
}

func GenerateSchema[T any]() openai.FunctionParameters {
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)

	param := openai.FunctionParameters{}
	b, _ := json.Marshal(schema)
	_ = json.Unmarshal(b, &param)

	return param
}
