package kube

import (
	"errors"
	"fmt"

	model "github.com/luoruofeng/Naval/model"

	"gopkg.in/yaml.v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// convertMapKeysToStrings 将map[interface{}]interface{}转化为map[string]interface{}
func convertMapKeysToStrings(obj interface{}) interface{} {

	switch t := obj.(type) {
	case map[interface{}]interface{}:
		m := map[string]interface{}{}

		for key, val := range t {

			m[fmt.Sprintf("%v", key)] = convertMapKeysToStrings(val)
		}
		return m
	case map[string]interface{}:
		m := map[string]interface{}{}

		for key, val := range t {

			m[fmt.Sprintf("%v", key)] = convertMapKeysToStrings(val)
		}
		return m
	case []interface{}:
		for i, val := range t {

			t[i] = convertMapKeysToStrings(val)
		}
	}

	return obj
}

// YmlToUnstructured 将yaml格式的字符串转换为unstructured.Unstructured对象
func YmlToUnstructured(ymlContent string) (*unstructured.Unstructured, error) {
	var unstructuredObj *unstructured.Unstructured = new(unstructured.Unstructured)
	var mobj map[string]interface{} = make(map[string]interface{})
	yamlData := []byte(ymlContent)
	err := yaml.Unmarshal(yamlData, mobj)
	if err != nil {
		return nil, err
	}
	r, ok := convertMapKeysToStrings(mobj).(map[string]interface{})
	if !ok {
		return nil, errors.New("将map[interface{}]interface{}转化为map[string]interface{}失败")
	}
	unstructuredObj.Object = r
	return unstructuredObj, nil
}

// 循环获取参数task中items并判断item是否有k8s yaml内容，如果有则获取kind和name放入slice中并返回
func GetK8sYamlKindAndName(task *model.Task) ([]string, []string, error) {
	var kindSlice []string
	var nameSlice []string
	if task.Items == nil {
		return nil, nil, errors.New("task中items为空")
	}
	for _, item := range task.Items {
		if item.K8SYamlContent != "" {
			// 获取kind
			var kind string
			var name string
			var mobj map[string]interface{} = make(map[string]interface{})
			yamlData := []byte(item.K8SYamlContent)
			yaml.Unmarshal(yamlData, mobj)
			r, ok := convertMapKeysToStrings(mobj).(map[string]interface{})
			if !ok {
				return nil, nil, errors.New("将map[interface{}]interface{}转化为map[string]interface{}失败")
			}
			kind = r["kind"].(string)
			name = r["metadata"].(map[string]interface{})["name"].(string)
			kindSlice = append(kindSlice, kind)
			nameSlice = append(nameSlice, name)
		}
	}
	return kindSlice, nameSlice, nil
}

// 根据传入的names循环修改参数task中items的metadata的name
func SetK8sYamlName(task *model.Task, names []string) error {
	for i, item := range task.Items {
		if item.K8SYamlContent != "" {
			var mobj map[string]interface{} = make(map[string]interface{})
			yamlData := []byte(item.K8SYamlContent)
			yaml.Unmarshal(yamlData, mobj)
			r, ok := convertMapKeysToStrings(mobj).(map[string]interface{})
			if !ok {
				return errors.New("将map[interface{}]interface{}转化为map[string]interface{}失败")
			}
			r["metadata"].(map[string]interface{})["name"] = names[i]
			yamlData, err := yaml.Marshal(r)
			if err != nil {
				return err
			}
			task.Items[i].K8SYamlContent = string(yamlData)
		}
	}
	return nil
}
