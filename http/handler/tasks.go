package handler

import (
	"errors"
	"net/http"

	"github.com/luoruofeng/Naval/model"
	"github.com/luoruofeng/Naval/srv"
	"go.uber.org/zap"
)

type TasksHandler struct {
	log     *zap.Logger
	taskSrv *srv.TaskSrv
}

func (*TasksHandler) Pattern() string {
	return "/tasks"
}

func NewTasksHandler(log *zap.Logger, taskSrv *srv.TaskSrv) *TasksHandler {
	return &TasksHandler{log: log, taskSrv: taskSrv}
}

func (h *TasksHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	uuid := r.Header.Get("X-Request-Id")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		result := struct {
			Message string `json:"message"`
			Error   error  `json:"error"`
		}{
			Message: "获取所有任务-失败",
			Error:   errors.New("HTTP Method not allowed"),
		}
		WriteResponseByJson(w, http.StatusMethodNotAllowed, result)
		return
	}

	tasks, err := h.taskSrv.GetAllTask()
	if err != nil {
		h.log.Error("获取所有任务-失败", zap.String("uuid", uuid), zap.Error(err))
		result := struct {
			Message string `json:"message"`
			Error   error  `json:"error"`
		}{
			Message: "获取所有任务-失败",
			Error:   err,
		}
		WriteResponseByJson(w, http.StatusInternalServerError, result)
		return
	}

	result := struct {
		Tasks []model.Task `json:"tasks"`
	}{
		Tasks: tasks,
	}
	WriteResponseByJson(w, http.StatusOK, result)
}
