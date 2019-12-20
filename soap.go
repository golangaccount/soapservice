package soapservice

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"
	"strconv"
	"strings"
)

//SOAPEnvelope Envelope节点
type SOAPEnvelope struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Envelope"`
	Header  *SOAPHeader
	Body    SOAPBody
}

//SOAPHeader Header节点
type SOAPHeader struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Header"`

	Header interface{}
}

//SOAPBody Body节点
type SOAPBody struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Body"`

	Fault   *SOAPFault  `xml:",omitempty"`
	Content interface{} `xml:",omitempty"`
}

//SOAPFault Fault节点
type SOAPFault struct {
	XMLName xml.Name `xml:"http://schemas.xmlsoap.org/soap/envelope/ Fault"`

	Code   string `xml:"faultcode,omitempty"`
	String string `xml:"faultstring,omitempty"`
	Actor  string `xml:"faultactor,omitempty"`
	Detail string `xml:"detail,omitempty"`
}

//UnmarshalXML 反序列化
func (b *SOAPBody) UnmarshalXML(d *xml.Decoder, start xml.StartElement) error {
	if b.Content == nil {
		return xml.UnmarshalError("Content must be a pointer to a struct")
	}

	var (
		token    xml.Token
		err      error
		consumed bool
	)

Loop:
	for {
		if token, err = d.Token(); err != nil {
			return err
		}

		if token == nil {
			break
		}

		switch se := token.(type) {
		case xml.StartElement:
			if consumed {
				return xml.UnmarshalError("Found multiple elements inside SOAP body; not wrapped-document/literal WS-I compliant")
			} else if se.Name.Space == "http://schemas.xmlsoap.org/soap/envelope/" && se.Name.Local == "Fault" {
				b.Fault = &SOAPFault{}
				b.Content = nil

				err = d.DecodeElement(b.Fault, &se)
				if err != nil {
					return err
				}

				consumed = true
			} else {
				if err = d.DecodeElement(b.Content, &se); err != nil {
					return err
				}

				consumed = true
			}
		case xml.EndElement:
			break Loop
		}
	}

	return nil
}

func (f *SOAPFault) Error() string {
	return f.String
}

//ActionInterface action转换函数
type ActionInterface interface {
	Action() map[string]string
}

//Soap soap协议
func Soap(Struct interface{}) func(http.ResponseWriter, *http.Request) {
	tp := reflect.TypeOf(Struct)
	if tp.Kind() != reflect.Ptr || tp.Elem().Kind() != reflect.Struct {
		panic("必须是struct指针")
	}

	if field, has := tp.Elem().FieldByName("Header"); has && (!(field.Type.Kind() == reflect.Ptr && field.Type.Elem().Kind() == reflect.Struct || field.Type.Kind() == reflect.Struct)) {
		panic("有Header字段时，必须为struct或struct指针")
	}

	if field, has := tp.Elem().FieldByName("User"); has && field.Type.Kind() != reflect.String {
		panic("有User字段时，必须为string")
	}

	if field, has := tp.Elem().FieldByName("Password"); has && field.Type.Kind() != reflect.String {
		panic("有Password字段时，必须为string")
	}
	actionmap := map[string]string{}
	if ai, pass := Struct.(ActionInterface); pass {
		actionmap = ai.Action()
	}

	return func(resp http.ResponseWriter, req *http.Request) {
		defer func() {
			fmt.Println(recover())
		}()
		defer req.Body.Close()
		instance := reflect.New(tp.Elem())
		inparm := []reflect.Value{}
		action := strings.Trim(strings.Replace(req.Header.Get("SOAPAction"), "http://tempuri.org/", "", -1), "\"")
		fmt.Println("action:", action)
		if v, has := actionmap[action]; has {
			action = v
		}
		reqparm := SOAPEnvelope{}
		if method, has := tp.MethodByName(action); has {
			if method.Type.NumIn() > 1 {
				intp := method.Type.In(1)
				var in reflect.Value
				if intp.Kind() == reflect.Ptr {
					in = reflect.New(intp.Elem())
					inparm = append(inparm, in)
					reqparm.Body.Content = in.Interface()
				} else {
					in = reflect.New(intp)
					inparm = append(inparm, in.Elem())
					reqparm.Body.Content = in.Interface()
				}
			}
			if field, has := tp.Elem().FieldByName("Header"); has {
				head := reflect.New(field.Type).Elem()
				instance.FieldByName("Header").Set(head)
				reqparm.Header.Header = head.Interface()
			}
			if user, password, has := req.BasicAuth(); has {
				if field, has := tp.Elem().FieldByName("User"); has && field.Type.Kind() == reflect.String {
					instance.FieldByName("User").SetString(user)
				}
				if field, has := tp.Elem().FieldByName("Password"); has && field.Type.Kind() == reflect.String {
					instance.FieldByName("Password").SetString(password)
				}
			}
			bts, err := ioutil.ReadAll(req.Body)
			fmt.Println(string(bts))
			if err != nil {
				fmt.Println("body err：", err.Error())
			}
			err = xml.Unmarshal(bts, &reqparm)
			if err != nil {
				fmt.Println("Unmarshal:", err.Error())
			}
			outs := instance.MethodByName(action).Call(inparm)
			respdata := SOAPEnvelope{}
			if len(outs) > 0 {
				respdata.Body.Content = outs[0].Interface()
			}
			bts, err = xml.Marshal(respdata)
			if err != nil {
				fmt.Println("Marshal:", err.Error())
			}
			resp.Header().Add("Content-Type", "text/xml; charset=utf-8")
			resp.Header().Add("Content-Length", strconv.Itoa(len(bts)))
			resp.WriteHeader(200)
			resp.Write(bts)
		}
	}
}
