# Phase 03: Conversation Persistence

This phase implements conversation save/load functionality and export capabilities. Users can save ongoing conversations to JSON files, resume them later, and export completed conversations to Markdown format. This is essential for the MVP's session persistence requirement.

## Tasks

- [x] Implement conversation saving in `pkg/v2/persistence/save.go`:
  - SaveConversation() function that takes Conversation and saves to JSON file
  - Generate filename from conversation ID and timestamp: `{id}_{timestamp}.json`
  - Create save directory if it doesn't exist (default: `~/.agentpipe/v2/conversations/`)
  - Use json.MarshalIndent for human-readable output
  - Handle file write errors with clear messages
  - Return saved file path
  - **Note:** Added ValidateConversation() function and DefaultSaveDir() helper

- [x] Implement conversation loading in `pkg/v2/persistence/load.go`:
  - LoadConversation() function that takes file path and returns Conversation
  - Parse JSON file and unmarshal to Conversation struct
  - Validate loaded conversation has required fields (ID, Messages, Started)
  - Return clear error for missing file, invalid JSON, or validation failure
  - LoadLatest() function that finds most recent conversation file in directory
  - ListConversations() that returns metadata (ID, date, message count, agents) for all saved conversations
  - **Note:** Added FindConversationByID() helper for resuming by ID prefix

- [x] Implement auto-save functionality in ConversationManager:
  - Add Persistence field to manager.Config with Enabled, SaveDir, SaveInterval fields
  - Add autoSave goroutine that triggers on interval and after each agent response
  - Subscribe to EventAgentDone in manager to trigger save
  - Debounce saves to avoid excessive disk writes
  - Log save operations without blocking conversation flow
  - **Note:** Implemented via setupAutoSave(), autoSaveLoop(), and triggerAutoSave() methods

- [x] Implement session resume in ConversationManager:
  - Resume() method that takes conversation ID or file path
  - Load conversation, restore Messages and Agents
  - Re-initialize adapters for loaded agents
  - Emit EventConversationStarted with resumed flag
  - Update conversation Status to "active"
  - Handle adapter initialization failures gracefully
  - **Note:** Added restoreConversation() helper and IsResumed() method

- [x] Implement Markdown export in `pkg/v2/persistence/export.go`:
  - ExportToMarkdown() function that takes Conversation and returns formatted string
  - Include conversation metadata header (ID, date, agents, message count)
  - Format each message with role, agent name, timestamp
  - Include metrics for agent messages (tokens, cost, duration)
  - Separate messages with horizontal rules
  - Include total summary at end (total tokens, total cost, duration)
  - SaveAsMarkdown() that writes export to file
  - **Note:** Added GenerateMarkdownFilename() helper

- [x] Write tests for persistence in `pkg/v2/persistence/persistence_test.go`:
  - TestSaveConversation: Save conversation, verify file exists and content valid
  - TestLoadConversation: Save then load, verify all fields match
  - TestLoadConversationMissingFile: Return error for missing file
  - TestLoadLatest: Save multiple, verify latest returned
  - TestListConversations: Save multiple, verify all listed with correct metadata
  - TestExportToMarkdown: Verify markdown output format
  - Use t.TempDir() for test directories
  - **Note:** Added 22 comprehensive tests covering all edge cases

- [x] Update the v2demo to demonstrate persistence:
  - Add --save flag to save conversation after completion
  - Add --resume flag to resume from conversation ID or "latest"
  - Add --export flag to export completed conversation to Markdown
  - Print save location when conversation saved
  - Print loaded conversation summary when resuming
  - **Note:** Added --list flag to list saved conversations

## Implementation Summary

**Files Created/Modified:**
- `pkg/v2/persistence/save.go` - Enhanced with validation and DefaultSaveDir()
- `pkg/v2/persistence/load.go` - New file with LoadConversation, LoadLatest, ListConversations, FindConversationByID
- `pkg/v2/persistence/export.go` - New file with ExportToMarkdown, SaveAsMarkdown, GenerateMarkdownFilename
- `pkg/v2/persistence/persistence_test.go` - New file with 22 comprehensive tests
- `pkg/v2/manager/manager.go` - Enhanced with PersistenceConfig, auto-save, Resume(), Save(), ExportToMarkdown()
- `cmd/v2demo/main.go` - Enhanced with --save, --resume, --export, --list flags

**All tests pass with race detection enabled.**
