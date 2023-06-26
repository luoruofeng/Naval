package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/luoruofeng/Naval/srv"
	"go.uber.org/zap"
)

type TaskHandler struct {
	log     *zap.Logger
	taskSrv *srv.TaskSrv
}

func (*TaskHandler) Pattern() string {
	return "/task"
}

func NewTaskHandler(log *zap.Logger, taskSrv *srv.TaskSrv) *TaskHandler {
	return &TaskHandler{log: log, taskSrv: taskSrv}
}

func (h *TaskHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	uuid := r.Header.Get("X-Request-Id")
	if r.Method != http.MethodPost && r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var message string

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		h.log.Error("Could not read request body", zap.String("uuid", uuid), zap.Error(err))
		http.Error(w, "Could not read request body", http.StatusInternalServerError)
		return
	}

	task, err := h.taskSrv.Unmarshal(body)
	if err != nil {
		h.log.Error("Could not parse YAML", zap.String("uuid", uuid), zap.Error(err))
		http.Error(w, "Could not parse YAML", http.StatusInternalServerError)
		return
	}

	if err := task.Verify(); err != nil {
		h.log.Info("task verify failed", zap.String("uuid", uuid), zap.Error(err))
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.log.Info("task verify success", zap.String("uuid", uuid), zap.Any("task", task))
	task.Uuid = uuid

	if r.Method == http.MethodPost {
		message = "创建任务成功"
		err = h.taskSrv.Add(*task)
		if err != nil {
			h.log.Error("创建任务失败", zap.Any("task_id", task.Id), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			result := struct {
				TaskId  string `json:"task_id"`
				Message string `json:"message"`
				Error   string `json:"error"`
			}{
				TaskId:  task.Id,
				Message: "Failed to create task",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(w).Encode(result)
			return
		}
	} else if r.Method == http.MethodPut {
		message = "更新任务成功"
		err = h.taskSrv.Update(*task)
		if err != nil {
			h.log.Error("更新任务失败", zap.Any("task", task), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			result := struct {
				TaskId  string `json:"task_id"`
				Message string `json:"message"`
				Error   string `json:"error"`
			}{
				TaskId:  task.Id,
				Message: "Failed to update task",
				Error:   err.Error(),
			}
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(w).Encode(result)
			return
		}
	}

	result := struct {
		TaskId  string `json:"task_id"`
		Message string `json:"message"`
	}{
		TaskId:  task.Id,
		Message: message,
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
