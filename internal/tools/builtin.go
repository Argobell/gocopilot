package tools

import ()

func RegisterBuiltinTools(registry *Registry, log interface{ Debug(format string, args ...interface{}); Info(format string, args ...interface{}); Error(format string, args ...interface{}) }) error {
	tools := []ToolDefinition{
		ReadFileDefinition,
		ListFilesDefinition,
		BashDefinition,
		EditFileDefinition,
		CodeSearchDefinition,
	}

	for _, tool := range tools {
		if err := registry.Register(tool); err != nil {
			log.Error("Failed to register tool %s: %v", tool.Name, err)
			return err
		}
		log.Debug("Registered tool: %s", tool.Name)
	}

	log.Info("Registered %d built-in tools", len(tools))
	return nil
}