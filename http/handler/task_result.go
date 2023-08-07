package handler

import (
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	mongo "github.com/luoruofeng/Naval/component/mongo/logic"
	"github.com/luoruofeng/Naval/model"
	"go.uber.org/zap"
)

type TaskResultHandler struct {
	log                *zap.Logger
	taskResultMongoSrv mongo.TaskResultMongoSrv
}

func (*TaskResultHandler) Pattern() string {
	return "/taskresult/{task_id}"
}

func NewTaskResultHandler(log *zap.Logger, taskResultMongoSrv mongo.TaskResultMongoSrv) *TaskResultHandler {
	return &TaskResultHandler{log: log, taskResultMongoSrv: taskResultMongoSrv}
}

func (h *TaskResultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	uuid := r.Header.Get("X-Request-Id")
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		result := struct {
			Message string `json:"message"`
			Error   error  `json:"error"`
		}{
			Message: "获取所有任务结果-失败",
			Error:   errors.New("HTTP Method not allowed"),
		}
		WriteResponseByJson(w, http.StatusMethodNotAllowed, result)
		return
	}

	vars := mux.Vars(r)
	taskId := vars["task_id"]
	results, err := h.taskResultMongoSrv.FindById(taskId)
	if err != nil {
		h.log.Error("获取所有任务结果-失败", zap.String("uuid", uuid), zap.Error(err))
		result := struct {
			Message string `json:"message"`
			Error   error  `json:"error"`
		}{
			Message: "获取所有任务结果-失败",
			Error:   err,
		}
		WriteResponseByJson(w, http.StatusInternalServerError, result)
		return
	}

	result := struct {
		Results []*model.TaskResult `json:"task_results"`
	}{
		Results: results,
	}
	WriteResponseByJson(w, http.StatusOK, result)
}
