// Copyright 2018 gf Author(https://gitee.com/johng/gf). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://gitee.com/johng/gf.
// 服务注册 + hook管理.

package ghttp

import (
    "errors"
    "strings"
    "reflect"
    "gitee.com/johng/gf/g/util/gutil"
)

// 绑定URI到操作函数/方法
// pattern的格式形如：/user/list, put:/user, delete:/user, post:/user@johng.cn
// 支持RESTful的请求格式，具体业务逻辑由绑定的处理方法来执行
func (s *Server)bindHandlerItem(pattern string, item *HandlerItem) error {
    if s.status == 1 {
        return errors.New("server handlers cannot be changed while running")
    }
    return s.setHandler(pattern, item)
}

// 通过映射数组绑定URI到操作函数/方法
func (s *Server)bindHandlerByMap(m HandlerMap) error {
    for p, h := range m {
        if err := s.bindHandlerItem(p, h); err != nil {
            return err
        }
    }
    return nil
}

// 将方法名称按照设定的规则转换为URI并附加到指定的URI后面
func (s *Server)appendMethodNameToUriWithPattern(pattern string, name string) string {
    // 检测域名后缀
    array := strings.Split(pattern, "@")
    // 分离URI(其实可能包含HTTP Method)
    uri := array[0]
    uri = strings.TrimRight(uri, "/") + "/"
    // 方法名中间存在大写字母，转换为小写URI地址以“-”号链接每个单词
    for i := 0; i < len(name); i++ {
        if i > 0 && gutil.IsLetterUpper(name[i]) {
            uri += "-"
        }
        uri += strings.ToLower(string(name[i]))
    }
    // 加上指定域名后缀
    if len(array) > 1 {
        uri += "@" + array[1]
    }
    return uri
}

// 注意该方法是直接绑定函数的内存地址，执行的时候直接执行该方法，不会存在初始化新的控制器逻辑
func (s *Server)BindHandler(pattern string, handler HandlerFunc) error {
    return s.bindHandlerItem(pattern, &HandlerItem{
        ctype : nil,
        fname : "",
        faddr : handler,
    })
}

// 绑定对象到URI请求处理中，会自动识别方法名称，并附加到对应的URI地址后面
// 需要注意对象方法的定义必须按照ghttp.HandlerFunc来定义
func (s *Server)BindObject(pattern string, obj interface{}) error {
    m := make(HandlerMap)
    v := reflect.ValueOf(obj)
    t := v.Type()
    for i := 0; i < v.NumMethod(); i++ {
        name  := t.Method(i).Name
        key   := s.appendMethodNameToUriWithPattern(pattern, name)
        m[key] = &HandlerItem{
            ctype : nil,
            fname : "",
            faddr : v.Method(i).Interface().(func(*Request)),
        }
    }
    return s.bindHandlerByMap(m)
}

// 绑定对象到URI请求处理中，会自动识别方法名称，并附加到对应的URI地址后面
// 第三个参数methods支持多个方法注册，多个方法以英文“,”号分隔，区分大小写
func (s *Server)BindObjectMethod(pattern string, obj interface{}, methods string) error {
    m := make(HandlerMap)
    for _, v := range strings.Split(methods, ",") {
        method := strings.TrimSpace(v)
        fval   := reflect.ValueOf(obj).MethodByName(method)
        if !fval.IsValid() {
            return errors.New("invalid method name:" + method)
        }
        key   := s.appendMethodNameToUriWithPattern(pattern, method)
        m[key] = &HandlerItem{
            ctype : nil,
            fname : "",
            faddr : fval.Interface().(func(*Request)),
        }
    }
    return s.bindHandlerByMap(m)
}

// 绑定对象到URI请求处理中，会自动识别方法名称，并附加到对应的URI地址后面
// 需要注意对象方法的定义必须按照ghttp.HandlerFunc来定义
func (s *Server)BindObjectRest(pattern string, obj interface{}) error {
    m := make(HandlerMap)
    v := reflect.ValueOf(obj)
    t := v.Type()
    for i := 0; i < v.NumMethod(); i++ {
        name := t.Method(i).Name
        if _, ok := s.methodsMap[strings.ToUpper(name)]; !ok {
            continue
        }
        key   := name + ":" + pattern
        m[key] = &HandlerItem{
            ctype : nil,
            fname : "",
            faddr : v.Method(i).Interface().(func(*Request)),
        }
    }
    return s.bindHandlerByMap(m)
}

// 绑定控制器，控制器需要实现gmvc.Controller接口
// 这种方式绑定的控制器每一次请求都会初始化一个新的控制器对象进行处理，对应不同的请求会话
func (s *Server)BindController(pattern string, c Controller) error {
    // 遍历控制器，获取方法列表，并构造成uri
    m := make(HandlerMap)
    v := reflect.ValueOf(c)
    t := v.Type()
    for i := 0; i < v.NumMethod(); i++ {
        name := t.Method(i).Name
        if name == "Init" || name == "Shut" {
            continue
        }
        key   := s.appendMethodNameToUriWithPattern(pattern, name)
        m[key] = &HandlerItem{
            ctype : v.Elem().Type(),
            fname : name,
            faddr : nil,
        }
    }
    return s.bindHandlerByMap(m)
}

// 绑定控制器(RESTFul)，控制器需要实现gmvc.Controller接口
// 方法会识别HTTP方法，并做REST绑定处理，例如：Post方法会绑定到HTTP POST的方法请求处理，Delete方法会绑定到HTTP DELETE的方法请求处理
// 因此只会绑定HTTP Method对应的方法，其他方法不会自动注册绑定
// 这种方式绑定的控制器每一次请求都会初始化一个新的控制器对象进行处理，对应不同的请求会话
func (s *Server)BindControllerRest(pattern string, c Controller) error {
    // 遍历控制器，获取方法列表，并构造成uri
    m := make(HandlerMap)
    v := reflect.ValueOf(c)
    t := v.Type()
    methods := make(map[string]bool)
    for _, v := range strings.Split(gHTTP_METHODS, ",") {
        methods[v] = true
    }
    // 如果存在与HttpMethod对应名字的方法，那么绑定这些方法
    for i := 0; i < v.NumMethod(); i++ {
        name := strings.ToUpper(t.Method(i).Name)
        if name == "Init" || name == "Shut" {
            continue
        }
        if _, ok := s.methodsMap[name]; !ok {
            continue
        }
        key   := name + ":" + pattern
        m[key] = &HandlerItem{
            ctype : v.Elem().Type(),
            fname : name,
            faddr : nil,
        }
    }
    return s.bindHandlerByMap(m)
}

// 这种方式绑定的控制器每一次请求都会初始化一个新的控制器对象进行处理，对应不同的请求会话
// 第三个参数methods支持多个方法注册，多个方法以英文“,”号分隔，不区分大小写
func (s *Server)BindControllerMethod(pattern string, c Controller, methods string) error {
    m    := make(HandlerMap)
    cval := reflect.ValueOf(c)
    for _, v := range strings.Split(methods, ",") {
        ctype  := reflect.ValueOf(c).Elem().Type()
        method := strings.TrimSpace(v)
        if !cval.MethodByName(method).IsValid() {
            return errors.New("invalid method name:" + method)
        }
        key    := s.appendMethodNameToUriWithPattern(pattern, method)
        m[key]  = &HandlerItem{
            ctype : ctype,
            fname : method,
            faddr : nil,
        }
    }
    return s.bindHandlerByMap(m)
}

// 绑定指定的hook回调函数, pattern参数同BindHandler，支持命名路由；hook参数的值由ghttp server设定，参数不区分大小写
func (s *Server)BindHookHandler(pattern string, hook string, handler HandlerFunc) error {
    //domain, method, uri, err := s.parsePatternForBindHookHandler(pattern)
    //if err != nil {
    //    return errors.New("invalid pattern")
    //}
    //var l *glist.List
    //if method == gDEFAULT_METHOD {
    //    for v, _ := range s.methodsMap {
    //        if v := s.hooksMap.GetWithDefault(s.handlerHookKey(domain, v, uri, hook), glist.New()); v != nil {
    //            l = v.(*glist.List)
    //        }
    //        l.PushBack(handler)
    //    }
    //} else {
    //    if v := s.hooksMap.GetWithDefault(s.handlerHookKey(domain, method, uri, hook), glist.New()); v == nil {
    //        l = v.(*glist.List)
    //    }
    //    l.PushBack(handler)
    //}
    return nil
}

// 通过map批量绑定回调函数
func (s *Server)BindHookHandlerByMap(pattern string, hookmap map[string]HandlerFunc) error {
    for k, v := range hookmap {
        if err := s.BindHookHandler(pattern, k, v); err != nil {
            return err
        }
    }
    return nil
}

// 构造用于hooksMap检索的键名
func (s *Server)handlerHookKey(domain, method, uri, hook string) string {
    return strings.ToUpper(hook) + "^" + s.handlerKey(domain, method, uri)
}
