package handler

import (
	"encoding/json"
	"io/ioutil"
	"net/http"

	"github.com/luoruofeng/Naval/model"
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

	var message string = ""
	if r.Method == http.MethodPost || r.Method == http.MethodPut {
		if r.Method == http.MethodPost {
			message = "创建任务"
		} else if r.Method == http.MethodPut {
			message = "更新任务"
		}
		if task.Type == model.Convert {
			message += "-转换成k8s任务"
		} else if task.Type == model.Create {
			message += "-k8s执行任务"
		} else {
			h.log.Error(message+"-失败-未设置Type无法分辨任务类型", zap.Any("task_id", task.Id), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			result := ErrorResponse{
				TaskId:  task.Id,
				Message: "Failed to create task",
				Error:   "创建/更新任务失败-无法分辨任务类型",
			}
			WriteResponseByJson(w, http.StatusInternalServerError, result)
			return
		}

		var err error
		if r.Method == http.MethodPost {
			err = h.taskSrv.Add(*task)
		} else if r.Method == http.MethodPut {
			if task.Type == model.Convert {
				err = h.taskSrv.UpdateConvert(*task)
			} else if task.Type == model.Create {
				err = h.taskSrv.Update(*task)
			}
		}

		if err != nil {
			h.log.Error(message+"-失败", zap.Any("task_id", task.Id), zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			result := ErrorResponse{
				TaskId:  task.Id,
				Message: "Failed to create task",
				Error:   err.Error(),
			}
			WriteResponseByJson(w, http.StatusInternalServerError, result)
			return
		}
	} else {
		h.log.Error(message+"-失败-未知的请求方法", zap.Any("task_id", task.Id), zap.Error(err))
		w.WriteHeader(http.StatusInternalServerError)
		result := struct {
			TaskId  string `json:"task_id"`
			Message string `json:"message"`
			Error   string `json:"error"`
		}{
			TaskId:  task.Id,
			Message: "Failed to create task",
			Error:   "创建/更新任务失败-未知的请求方法",
		}
		WriteResponseByJson(w, http.StatusInternalServerError, result)
		return
	}

	result := SuccessResponse{
		TaskId:  task.Id,
		Message: message + "-成功",
	}
	WriteResponseByJson(w, http.StatusOK, result)
}

type SuccessResponse struct {
	TaskId  string `json:"task_id"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	TaskId  string `json:"task_id"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

func WriteResponseByJson(w http.ResponseWriter, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(data)
}
