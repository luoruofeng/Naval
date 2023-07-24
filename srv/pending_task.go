package srv

import (
	"errors"

	m "github.com/luoruofeng/Naval/model"
)

func (ts *TaskSrv) DeleteNil() {
	ts.lock.Lock()
	defer ts.lock.Unlock()

	var result []*m.Task = make([]*m.Task, 0)
	for _, elem := range ts.pendingTasks {
		if elem != nil {
			result = append(result, elem)
		}
	}
	ts.pendingTasks = result
}

func (ts *TaskSrv) WalkPendingTasks(p func(i int, t *m.Task) (bool, error)) error {
	ts.lock.Lock()
	defer ts.lock.Unlock()
	for i, task := range ts.pendingTasks {
		if task == nil {
			continue
		}
		b, err := p(i, task)
		if err != nil {
			return err
		}
		if b {
			return nil
		}
	}
	return nil
}

func (ts *TaskSrv) AddPendingTask(t m.Task) error {
	err := ts.WalkPendingTasks(func(i int, task *m.Task) (bool, error) {
		if task.Id == t.Id {
			return true, errors.New("打算新增到pendingtasks中的任务id已经存在")
		}
		return false, nil
	})

	if err != nil {
		return err
	}

	ts.lock.Lock()
	ts.pendingTasks = append(ts.pendingTasks, &t)
	ts.lock.Unlock()
	return nil
}

func (ts *TaskSrv) DeletePendingTask(id string) error {
	return ts.WalkPendingTasks(func(i int, t *m.Task) (bool, error) {
		if t.Id == id {
			ts.pendingTasks = append(ts.pendingTasks[:i], ts.pendingTasks[i+1:]...)
			return true, nil
		}
		return false, nil
	})
}

func (ts *TaskSrv) UpdatePendingTask(task m.Task) error {
	return ts.WalkPendingTasks(func(i int, t *m.Task) (bool, error) {
		if t.Id == task.Id {
			ts.pendingTasks[i] = &task
			return true, nil
		}
		return false, nil
	})
}
