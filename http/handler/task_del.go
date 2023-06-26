package handler

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/luoruofeng/Naval/srv"
	"go.uber.org/zap"
)

type TaskDelHandler struct {
	log     *zap.Logger
	taskSrv *srv.TaskSrv
}

func (*TaskDelHandler) Pattern() string {
	return "/task/{id}"
}

func NewTaskDelHandler(log *zap.Logger, taskSrv *srv.TaskSrv) *TaskDelHandler {
	return &TaskDelHandler{log: log, taskSrv: taskSrv}
}

func (h *TaskDelHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()
	vars := mux.Vars(r)
	id := vars["id"]
	if err := h.taskSrv.Delete(id); err != nil {
		h.log.Error("删除任务失败", zap.Any("task_id", id), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)

		result := struct {
			TaskId  string `json:"task_id"`
			Message string `json:"message"`
			Error   string `json:"error"`
		}{
			TaskId:  id,
			Message: "Task deletion failed",
			Error:   err.Error(),
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(result)
		return
	}

	result := struct {
		TaskId  string `json:"task_id"`
		Message string `json:"message"`
	}{
		TaskId:  id,
		Message: "Task has been successfully deleted",
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(result)
}
