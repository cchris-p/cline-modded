package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/cline/cli/pkg/cli/display"
	"github.com/cline/cli/pkg/cli/global"
	"github.com/cline/cli/pkg/cli/sqlite"
	"github.com/cline/cli/pkg/common"
	"github.com/cline/cli/pkg/cli/types"
	"github.com/cline/grpc-go/cline"
)

type DeleteTasksSummary struct {
	RequestedTaskIDs     []string `json:"requested_task_ids"`
	HistoryFilePath      string   `json:"history_file_path"`
	HistoryItemsRemoved  int      `json:"history_items_removed"`
	HistoryItemsNotFound []string `json:"history_items_not_found"`
	TaskDirsDeleted      []string `json:"task_dirs_deleted"`
	TaskDirsMissing      []string `json:"task_dirs_missing"`
	FolderLocksRemoved   int      `json:"folder_locks_removed"`
	TasksDirRemoved      bool     `json:"tasks_dir_removed"`
	CheckpointsDirRemoved bool    `json:"checkpoints_dir_removed"`
}

func getTaskHistoryFilePath() (string, error) {
	if global.Config == nil {
		return "", fmt.Errorf("global config not initialized")
	}
	return filepath.Join(global.Config.ConfigPath, common.SETTINGS_SUBFOLDER, "state", "taskHistory.json"), nil
}

func getTasksBaseDir() (string, error) {
	if global.Config == nil {
		return "", fmt.Errorf("global config not initialized")
	}
	return filepath.Join(global.Config.ConfigPath, common.SETTINGS_SUBFOLDER, "tasks"), nil
}

func getCheckpointsBaseDir() (string, error) {
	if global.Config == nil {
		return "", fmt.Errorf("global config not initialized")
	}
	return filepath.Join(global.Config.ConfigPath, common.SETTINGS_SUBFOLDER, "checkpoints"), nil
}

func atomicWriteFile(filePath string, data []byte) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(filePath)+".tmp.*")
	if err != nil {
		return err
	}

	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpPath, filePath)
}

func readTaskHistoryFromDisk(filePath string) ([]types.HistoryItem, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return []types.HistoryItem{}, nil
		}
		return nil, fmt.Errorf("failed to read task history: %w", err)
	}

	var historyItems []types.HistoryItem
	if err := json.Unmarshal(data, &historyItems); err != nil {
		return nil, fmt.Errorf("failed to parse task history: %w", err)
	}

	return historyItems, nil
}

func writeTaskHistoryToDisk(filePath string, items []types.HistoryItem) error {
	data, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("failed to serialize task history: %w", err)
	}
	if err := atomicWriteFile(filePath, data); err != nil {
		return fmt.Errorf("failed to write task history: %w", err)
	}
	return nil
}

func removeTaskFolderLocks(taskIDs []string) (int, error) {
	lockManager, err := sqlite.NewLockManager(global.Config.ConfigPath)
	if err != nil {
		return 0, nil
	}
	defer lockManager.Close()

	removed := 0
	for _, id := range taskIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}

		targets := []string{
			filepath.ToSlash(filepath.Join(global.Config.ConfigPath, common.SETTINGS_SUBFOLDER, "tasks", id)),
			"~/.cline/data/tasks/" + id,
		}

		for _, t := range targets {
			n, err := lockManager.RemoveFolderLock(t)
			if err != nil {
				continue
			}
			removed += n
		}
	}

	return removed, nil
}

func DeleteTasksFromDisk(taskIDs []string) (*DeleteTasksSummary, error) {
	historyFilePath, err := getTaskHistoryFilePath()
	if err != nil {
		return nil, err
	}
	tasksBaseDir, err := getTasksBaseDir()
	if err != nil {
		return nil, err
	}
	checkpointsBaseDir, err := getCheckpointsBaseDir()
	if err != nil {
		return nil, err
	}

	summary := &DeleteTasksSummary{
		RequestedTaskIDs: taskIDs,
		HistoryFilePath:  historyFilePath,
	}

	historyItems, err := readTaskHistoryFromDisk(historyFilePath)
	if err != nil {
		return nil, err
	}

	requested := map[string]struct{}{}
	for _, id := range taskIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		requested[id] = struct{}{}
	}

	filtered := make([]types.HistoryItem, 0, len(historyItems))
	for _, item := range historyItems {
		if _, ok := requested[item.Id]; ok {
			summary.HistoryItemsRemoved++
			continue
		}
		filtered = append(filtered, item)
	}

	for id := range requested {
		found := false
		for _, item := range historyItems {
			if item.Id == id {
				found = true
				break
			}
		}
		if !found {
			summary.HistoryItemsNotFound = append(summary.HistoryItemsNotFound, id)
		}
	}

	if err := writeTaskHistoryToDisk(historyFilePath, filtered); err != nil {
		return nil, err
	}

	for id := range requested {
		taskDir := filepath.Join(tasksBaseDir, id)
		if _, err := os.Stat(taskDir); err != nil {
			if os.IsNotExist(err) {
				summary.TaskDirsMissing = append(summary.TaskDirsMissing, taskDir)
				continue
			}
			return nil, fmt.Errorf("failed to stat task dir %s: %w", taskDir, err)
		}

		if err := os.RemoveAll(taskDir); err != nil {
			return nil, fmt.Errorf("failed to remove task dir %s: %w", taskDir, err)
		}
		summary.TaskDirsDeleted = append(summary.TaskDirsDeleted, taskDir)
	}

	if global.Config != nil {
		removed, _ := removeTaskFolderLocks(taskIDs)
		summary.FolderLocksRemoved = removed
	}

	if len(filtered) == 0 {
		if err := os.RemoveAll(tasksBaseDir); err == nil {
			summary.TasksDirRemoved = true
		}
		if err := os.RemoveAll(checkpointsBaseDir); err == nil {
			summary.CheckpointsDirRemoved = true
		}
	}

	return summary, nil
}

func GetTaskHistoryIDsFromDisk() ([]string, error) {
	filePath, err := getTaskHistoryFilePath()
	if err != nil {
		return nil, err
	}
	historyItems, err := readTaskHistoryFromDisk(filePath)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(historyItems))
	for _, item := range historyItems {
		if strings.TrimSpace(item.Id) == "" {
			continue
		}
		ids = append(ids, item.Id)
	}
	return ids, nil
}

// ListTasksFromDisk reads task history directly from disk
func ListTasksFromDisk() error {
	filePath, err := getTaskHistoryFilePath()
	if err != nil {
		return err
	}

	historyItems, err := readTaskHistoryFromDisk(filePath)
	if err != nil {
		return err
	}

	if len(historyItems) == 0 {
		fmt.Println("No task history found.")
		return nil
	}

	// Sort by timestamp ascending (oldest first, newest last)
	sort.Slice(historyItems, func(i, j int) bool {
		return historyItems[i].Ts < historyItems[j].Ts
	})

	// Convert to protobuf TaskItem format for rendering
	tasks := make([]*cline.TaskItem, len(historyItems))
	for i, item := range historyItems {
		tasks[i] = &cline.TaskItem{
			Id:          item.Id,
			Task:        item.Task,
			Ts:          item.Ts,
			IsFavorited: item.IsFavorited,
			Size:        item.Size,
			TotalCost:   item.TotalCost,
			TokensIn:    item.TokensIn,
			TokensOut:   item.TokensOut,
			CacheWrites: item.CacheWrites,
			CacheReads:  item.CacheReads,
		}
	}

	// Use existing renderer
	renderer := display.NewRenderer(global.Config.OutputFormat)
	return renderer.RenderTaskList(tasks)
}
