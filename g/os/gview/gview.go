// Copyright 2017 gf Author(https://gitee.com/johng/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://gitee.com/johng/gf.

// 视图管理.
package gview

import (
    "sync"
    "bytes"
    "errors"
    "html/template"
    "gitee.com/johng/gf/g/container/gmap"
    "gitee.com/johng/gf/g/encoding/ghash"
    "gitee.com/johng/gf/g/util/gconv"
    "gitee.com/johng/gf/g/os/gspath"
    "gitee.com/johng/gf/g/os/gfcache"
)

// 视图对象
type View struct {
    mu         sync.RWMutex
    paths      *gspath.SPath           // 模板查找目录(绝对路径)
    funcmap    map[string]interface{}  // FuncMap
    delimiters []string                // 模板变量分隔符号
}

// 输出到模板页面时保留HTML标签原意，不做自动escape处理
type HTML    = template.HTML

// 模板变量
type Params  = map[string]interface{}

// 函数映射表
type FuncMap = map[string]interface{}

// 视图表
var viewMap = gmap.NewStringInterfaceMap()

// 默认的视图对象
var viewObj = Get(".")

// 直接解析模板内容，返回解析后的内容
func ParseContent(content string, params map[string]interface{}) ([]byte, error) {
    return viewObj.ParseContent(content, params)
}

// 获取或者创建一个视图对象
func Get(path string) *View {
    if r := viewMap.Get(path); r != nil {
        return r.(*View)
    }
    v := New(path)
    viewMap.Set(path, v)
    return v
}

// 生成一个视图对象
func New(path string) *View {
    s := gspath.New()
    s.Set(path)
    view := &View {
        paths      : s,
        funcmap    : make(map[string]interface{}),
        delimiters : make([]string, 2),
    }
    view.SetDelimiters("{{", "}}")
    // 内置方法
    view.BindFunc("include", view.funcInclude)
    return view
}

// 设置模板目录绝对路径
func (view *View) SetPath(path string) error {
    return view.paths.Set(path)
}

// 添加模板目录搜索路径
func (view *View) AddPath(path string) error {
    return view.paths.Add(path)
}

// 解析模板，返回解析后的内容
func (view *View) Parse(file string, params map[string]interface{}, funcmap...map[string]interface{}) ([]byte, error) {
    path := view.paths.Search(file)
    if path == "" {
        return nil, errors.New("tpl \"" + file + "\" not found")
    }
    content := gfcache.GetContents(path)
    // 执行模板解析，互斥锁主要是用于funcmap
    view.mu.RLock()
    defer view.mu.RUnlock()
    buffer := bytes.NewBuffer(nil)
    tplobj := template.New(path).Delims(view.delimiters[0], view.delimiters[1]).Funcs(view.funcmap)
    if len(funcmap) > 0 {
        tplobj = tplobj.Funcs(funcmap[0])
    }
    if tpl, err := tplobj.Parse(content); err != nil {
        return nil, err
    } else {
        if err := tpl.Execute(buffer, params); err != nil {
            return nil, err
        }
    }
    return buffer.Bytes(), nil
}

// 直接解析模板内容，返回解析后的内容
func (view *View) ParseContent(content string, params map[string]interface{}, funcmap...map[string]interface{}) ([]byte, error) {
    view.mu.RLock()
    defer view.mu.RUnlock()
    name   := gconv.String(ghash.BKDRHash64([]byte(content)))
    buffer := bytes.NewBuffer(nil)
    tplobj := template.New(name).Delims(view.delimiters[0], view.delimiters[1]).Funcs(view.funcmap)
    if len(funcmap) > 0 {
        tplobj = tplobj.Funcs(funcmap[0])
    }
    if tpl, err := tplobj.Parse(content); err != nil {
        return nil, err
    } else {
        if err := tpl.Execute(buffer, params); err != nil {
            return nil, err
        }
    }
    return buffer.Bytes(), nil
}

// 设置模板变量解析分隔符号
func (view *View) SetDelimiters(left, right string) {
    view.delimiters[0] = left
    view.delimiters[1] = right
}

// 绑定自定义函数，该函数是全局有效，即调用之后每个线程都会生效，因此有并发安全控制
func (view *View) BindFunc(name string, function interface{}) {
    view.mu.Lock()
    view.funcmap[name] = function
    view.mu.Unlock()
}

// 模板内置方法：include
func (view *View) funcInclude(file string, data...map[string]interface{}) template.HTML {
    var m map[string]interface{} = nil
    if len(data) > 0 {
        m = data[0]
    }
    content, err := view.Parse(file, m)
    if err != nil {
        return template.HTML(err.Error())
    }
    return template.HTML(content)
}
