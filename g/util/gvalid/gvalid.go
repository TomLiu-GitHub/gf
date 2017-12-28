// 通用数据验证工具
// 本来打算取名gvalidator的，名字太长了，缩写一下
/*
参考：https://laravel.com/docs/5.5/validation#available-validation-rules
规则如下：
required           格式：required                      说明：必需参数
required_if        格式：required_if:field,value,...   说明：必需参数(当给定字段值与所给任意值相等时)
required_with      格式：required_with:foo,bar,...     说明：必需参数(当所给定任意字段值不为空时)
required_with_all  格式：required_with_all:foo,bar,... 说明：必须参数(当所给定所有字段值都不为空时)
date               格式：date                          说明：参数为常用日期类型，格式：2006-01-02, 20060102, 2006.01.02
date_format        格式：date_format:format            说明：判断日期是否为指定的日期格式，format为Go日期格式(可以包含时间)
email              格式：email                         说明：EMAIL邮箱地址
phone              格式：phone                         说明：手机号
telephone          格式：telephone                     说明：国内座机电话号码，"XXXX-XXXXXXX"、"XXXX-XXXXXXXX"、"XXX-XXXXXXX"、"XXX-XXXXXXXX"、"XXXXXXX"、"XXXXXXXX"
passport           格式：passport                      说明：通用帐号规则(字母开头，只能包含字母、数字和下划线，长度在6~18之间)
password           格式：password                      说明：通用密码(任意可见字符，长度在6~18之间)
password2          格式：password2                     说明：中等强度密码(在弱密码的基础上，必须包含大小写字母和数字)
password3          格式：password3                     说明：强等强度密码(在弱密码的基础上，必须包含大小写字母、数字和特殊字符)
postcode           格式：id_number                     说明：中国邮政编码
id_number          格式：id_number                     说明：公民身份证号码
qq                 格式：qq                            说明：腾讯QQ号码
ip                 格式：ip                            说明：IP地址(IPv4)
mac                格式：mac                           说明：MAC地址
url                格式：url                           说明：URL
length             格式：length:min,max                说明：参数长度为min到max
min_length         格式：min_length:min                说明：参数长度最小为min
max_length         格式：max_length:max                说明：参数长度最大为max
between            格式：between:min,max               说明：参数大小为min到max
min                格式：min:min                       说明：参数最小为min
max                格式：max:max                       说明：参数最大为max
json               格式：json                          说明：判断数据格式为JSON
xml                格式：xml                           说明：判断数据格式为XML
integer            格式：integer                       说明：整数
float              格式：float                         说明：浮点数
boolean            格式：boolean                       说明：布尔值(1,true,on,yes:true | 0,false,off,no,"":false)
same               格式：same:field                    说明：参数值必需与field参数的值相同
different          格式：different:field               说明：参数值不能与field参数的值相同
in                 格式：in:foo,bar,...                说明：参数值应该在foo,bar,...中
not_in             格式：not_in:foo,bar,...            说明：参数值不应该在foo,bar,...中
regex              格式：regex:pattern                 说明：参数值应当满足正则匹配规则pattern(使用preg_match判断)
*/
package gvalid

import (
    "strings"
    "regexp"
    "strconv"
    "gitee.com/johng/gf/g/os/gtime"
    "gitee.com/johng/gf/g/util/gregx"
    "gitee.com/johng/gf/g/encoding/gjson"
)

// 默认规则校验错误消息(可以通过接口自定义错误消息)
var defaultMessages = map[string]string {
    "required"          : "字段不能为空",
    "required_if"       : "字段不能为空",
    "required_with"     : "字段不能为空",
    "required_with_all" : "字段不能为空",
    "date"              : "日期格式不正确",
    "email"             : "邮箱地址格式不正确",
    "phone"             : "手机号码格式不正确",
    "telephone"         : "电话号码格式不正确",
    "passport"          : "账号格式不合法，必需以字母开头，只能包含字母、数字和下划线，长度在6~18之间",
    "password"          : "密码格式不合法，密码格式为任意6-18位的可见字符",
    "password2"         : "密码格式不合法，密码格式为任意6-18位的可见字符，必须包含大小写字母和数字",
    "password3"         : "密码格式不合法，密码格式为任意6-18位的可见字符，必须包含大小写字母、数字和特殊字符",
    "postcode"          : "邮政编码不正确",
    "id_number"         : "身份证号码不正确",
    "qq"                : "QQ号码格式不正确",
    "ip"                : "IP地址格式不正确",
    "mac"               : "MAC地址格式不正确",
    "url"               : "URL地址格式不正确",
    "length"            : "字段长度为:min到:max个字符",
    "min_length"        : "字段最小长度为:min",
    "max_length"        : "字段最大长度为:max",
    "between"           : "字段大小为:min到:max",
    "min"               : "字段最小值为:min",
    "max"               : "字段最大值为:max",
    "json"              : "字段应当为JSON格式",
    "xml"               : "字段应当为XML格式",
    "array"             : "字段应当为数组",
    "integer"           : "字段应当为整数",
    "float"             : "字段应当为浮点数",
    "boolean"           : "字段应当为布尔值",
    "same"              : "字段值不合法",
    "different"         : "字段值不合法",
    "in"                : "字段值不合法",
    "not_in"            : "字段值不合法",
    "regex"             : "字段值不合法",
}

// 检测一条数据的规则，其中values参数为非必须参数，可以传递所有的校验参数进来，进行多参数对比(部分校验规则需要)
func CheckRule(value, rule string, values...map[string]string) map[string]string {
    msgs   := make(map[string]string)
    params := make(map[string]string)
    if len(values) > 0 {
        params = values[0]
    }
    items  := strings.Split(strings.TrimSpace(rule), "|")
    for _, item := range items {
        reg, _  := regexp.Compile(`^(\w+):{0,1}(.*)`)
        results := reg.FindStringSubmatch(item)
        rulekey := results[1]
        ruleval := results[2]
        match   := false
        switch rulekey {
            // 必须字段
            case "required":
                match = !(value == "")
                break

            // 必须字段(当给定字段值与所给任意值相等时)
            case "required_if":
                required := false
                array    := strings.Split(strings.TrimSpace(ruleval), ",")
                // 必须为偶数，才能是键值对匹配
                if len(array)%2 == 0 {
                    for i := 0; i < len(array); {
                        tk := array[i]
                        tv := array[i + 1]
                        if v, ok := params[tk]; ok {
                            if strings.Compare(tv, v) == 0 {
                                required = true
                                break
                            }
                        }
                        i += 2
                    }
                }
                if required {
                    match = !(value == "")
                } else {
                    match = true
                }
                break

            // 必须字段(当所给定任意字段值不为空时)
            case "required_with":
                required := false
                array    := strings.Split(strings.TrimSpace(ruleval), ",")
                for i := 0; i < len(array); i++ {
                    if v, ok := params[array[i]]; ok {
                        if v != "" {
                            required = true
                            break
                        }
                    }
                }
                if required {
                    match = !(value == "")
                } else {
                    match = true
                }
                break

            // 必须字段(当所给定所有字段值都不为空时)
            case "required_with_all":
                required := true
                array    := strings.Split(strings.TrimSpace(ruleval), ",")
                for i := 0; i < len(array); i++ {
                    if v, ok := params[array[i]]; ok {
                        if v == "" {
                            required = false
                            break
                        }
                    }
                }
                if required {
                    match = !(value == "")
                } else {
                    match = true
                }
                break

            // 日期格式，
            case "date":
                for _, v := range []string{"2006-01-02", "20060102", "2006.01.02"} {
                    if _, err := gtime.StrToTime(value, v); err == nil {
                        match = true
                        break
                    }
                }
                break

            // 日期格式，需要给定日期格式
            case "date_format":
                if _, err := gtime.StrToTime(value, ruleval); err == nil {
                    match = true
                }
                break

            // 两字段值应相同(非敏感字符判断，非类型判断)
            case "same":
                if v, ok := params[ruleval]; ok {
                    if strings.Compare(value, v) == 0 {
                        match = true
                    }
                }
                break

            // 两字段值不应相同(非敏感字符判断，非类型判断)
            case "different":
                match = true
                if v, ok := params[ruleval]; ok {
                    if strings.Compare(value, v) == 0 {
                        match = false
                    }
                }
                break

            // 字段值应当在指定范围中
            case "in":
                array := strings.Split(strings.TrimSpace(ruleval), ",")
                for _, v := range array {
                    if strings.Compare(value, strings.TrimSpace(v)) == 0 {
                        match = true
                        break
                    }
                }
                break

            // 字段值不应当在指定范围中
            case "not_in":
                match  = true
                array := strings.Split(strings.TrimSpace(ruleval), ",")
                for _, v := range array {
                    if strings.Compare(value, strings.TrimSpace(v)) == 0 {
                        match = false
                        break
                    }
                }
                break

            // 自定义正则判断
            //case "regex":
            //    $ruleMatch = @preg_match($ruleAttr, $value) ? true : false;
            //    break

            /*
             * 验证所给手机号码是否符合手机号的格式.
             * 移动：134、135、136、137、138、139、150、151、152、157、158、159、182、183、184、187、188、178(4G)、147(上网卡)；
             * 联通：130、131、132、155、156、185、186、176(4G)、145(上网卡)、175；
             * 电信：133、153、180、181、189 、177(4G)；
             * 卫星通信：  1349
             * 虚拟运营商：170、173
             */
            case "phone":
                match = gregx.IsMatchString(`^13[\d]{9}$|^14[5,7]{1}\d{8}$|^15[^4]{1}\d{8}$|^17[0,3,5,6,7,8]{1}\d{8}$|^18[\d]{9}$`, value)
                break

            // 国内座机电话号码："XXXX-XXXXXXX"、"XXXX-XXXXXXXX"、"XXX-XXXXXXX"、"XXX-XXXXXXXX"、"XXXXXXX"、"XXXXXXXX"
            case "telephone":
                match = gregx.IsMatchString(`^((\d{3,4})|\d{3,4}-)?\d{7,8}$`, value)
                break

            // 腾讯QQ号，从10000开始
            case "qq":
                match = gregx.IsMatchString(`^[1-9][0-9]{4,}$`, value)
                break

                // 中国邮政编码
            case "postcode":
                match = gregx.IsMatchString(`^[1-9]\d{5}$`, value)
                break

            /*
                公民身份证号
                xxxxxx yyyy MM dd 375 0     十八位
                xxxxxx   yy MM dd  75 0     十五位

                地区：[1-9]\d{5}
                年的前两位：(18|19|([23]\d))      1800-2399
                年的后两位：\d{2}
                月份：((0[1-9])|(10|11|12))
                天数：(([0-2][1-9])|10|20|30|31) 闰年不能禁止29+

                三位顺序码：\d{3}
                两位顺序码：\d{2}
                校验码：   [0-9Xx]

                十八位：^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$
                十五位：^[1-9]\d{5}\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{2}$

                总：
                (^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$)|(^[1-9]\d{5}\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{2}$)
             */
            case "id_number":
                match = gregx.IsMatchString(`(^[1-9]\d{5}(18|19|([23]\d))\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$)|(^[1-9]\d{5}\d{2}((0[1-9])|(10|11|12))(([0-2][1-9])|10|20|30|31)\d{2}$)`, value)
                break

            // 通用帐号规则(字母开头，只能包含字母、数字和下划线，长度在6~18之间)
            case "passport":
                match = gregx.IsMatchString(`^[a-zA-Z]{1}\w{5,17}$`, value)
                break

            // 通用密码(任意可见字符，长度在6~18之间)
            case "password":
                match = gregx.IsMatchString(`^[\w\S]{6,18}$`, value)
                break

            // 中等强度密码(在弱密码的基础上，必须包含大小写字母和数字)
            case "password2":
                if gregx.IsMatchString(`^[\w\S]{6,18}$`, value)  && gregx.IsMatchString(`[a-z]+`, value) && gregx.IsMatchString(`[A-Z]+`, value) && gregx.IsMatchString(`\d+`, value) {
                    match = true
                }
                break

                // 强等强度密码(在弱密码的基础上，必须包含大小写字母、数字和特殊字符)
            case "password3":
                if gregx.IsMatchString(`^[\w\S]{6,18}$`, value) && gregx.IsMatchString(`[a-z]+`, value) && gregx.IsMatchString(`[A-Z]+`, value) && gregx.IsMatchString(`\d+`, value) && gregx.IsMatchString(`\S+`, value) {
                    match = true
                }
                break

                // 长度范围
            case "length":
                array := strings.Split(strings.TrimSpace(ruleval), ",")
                min   := 0
                max   := 0
                if len(array) > 0 {
                    if v, err := strconv.Atoi(array[0]); err == nil {
                        min = v
                    }
                }
                if len(array) > 1 {
                    if v, err := strconv.Atoi(array[1]); err == nil {
                        max = v
                    }
                }
                if len(value) >= min && len(value) <= max {
                    match = true
                }
                break

            // 最小长度
            case "min_length":
                if min, err := strconv.Atoi(strings.TrimSpace(ruleval)); err == nil {
                    if len(value) >= min {
                        match = true
                    }
                }
                break

            // 最大长度
            case "max_length":
                if max, err := strconv.Atoi(strings.TrimSpace(ruleval)); err == nil {
                    if len(value) <= max {
                        match = true
                    }
                }
                break

            // 大小范围
            case "between":
                array := strings.Split(strings.TrimSpace(ruleval), ",")
                min   := 0
                max   := 0
                if len(array) > 0 {
                    if v, err := strconv.Atoi(array[0]); err == nil {
                        min = v
                    }
                }
                if len(array) > 1 {
                    if v, err := strconv.Atoi(array[1]); err == nil {
                        max = v
                    }
                }
                if v, err := strconv.Atoi(value); err == nil {
                    if v >= min && v <= max {
                        match = true
                    }
                }
                break

            // 最小值
            case "min":
                if min, err := strconv.Atoi(strings.TrimSpace(ruleval)); err == nil {
                    if v, err := strconv.Atoi(value); err == nil {
                        if v >= min {
                            match = true
                        }
                    }
                }
                break

            // 最大值
            case "max":
                if max, err := strconv.Atoi(strings.TrimSpace(ruleval)); err == nil {
                    if v, err := strconv.Atoi(value); err == nil {
                        if v <= max {
                            match = true
                        }
                    }
                }
                break

            // json
            case "json":
                if _, err := gjson.Decode([]byte(value)); err == nil {
                    match = true
                }
                break

            //// xml
            //case "xml":
            //    $checkResult = @Lib_XmlParser::isXml($value);
            //    $ruleMatch   = ($checkResult !== null && $checkResult !== false);
            //    break

            // 整数
            case "integer":
                if _, err := strconv.Atoi(value); err == nil {
                    match = true
                }
                break

            // 小数
            case "float":
                if _, err := strconv.ParseFloat(value, 10); err == nil {
                    match = true
                }
                break

            // 布尔值(1,true,on,yes:true | 0,false,off,no,"":false)
            case "boolean":
                if value != "" && value != "0" && value != "false" && value != "off" && value != "no" {
                    match = true
                }
                break

            // 邮件
            case "email":
                match = gregx.IsMatchString(`^[a-zA-Z0-9_-]+@[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+$`, value)
                break

            // URL
            case "url":
                match = gregx.IsMatchString(`^(?:([A-Za-z]+):)?(\/{0,3})([0-9.\-A-Za-z]+)(?::(\d+))?(?:\/([^?#]*))?(?:\?([^#]*))?(?:#(.*))?$`, value)
                break

            // IP
            case "ip":
                match = gregx.IsMatchString(`^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$`, value)
                break

            // MAC地址
            case "mac":
                match = gregx.IsMatchString(`^([0-9A-Fa-f]{2}-){5}[0-9A-Fa-f]{2}$`, value)
                break

            default:
                msgs[rulekey] = "Invalid rule name:" + rulekey
                break
        }

        // 错误消息整合
        if !match {
            msgs[rulekey] = defaultMessages[rulekey]
        }
    }
    if len(msgs) > 0 {
        return msgs
    }
    return nil
}