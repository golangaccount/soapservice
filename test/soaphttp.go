package main

import (
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/golangaccount/soapservice"
)

func main() {
	http.HandleFunc("/soap", soapservice.Soap(&Login{}))
	http.ListenAndServe("0.0.0.0:12300", http.DefaultServeMux)
}

type Login struct {
	//Header interface{}
}

type PutDataBySqhReq struct {
	XMLName  xml.Name `xml:"http://tempuri.org/ putDataBySqh"`
	Sqh      string   `xml:"sqh"`
	Data     string   `xml:"data"`
	User     string   `xml:"user"`
	Password string   `xml:"password"`
}

type PutDataBySqhResponse struct {
	XMLName            xml.Name `xml:"http://tempuri.org/ putDataBySqhResponse"`
	PutDataBySqhResult string   `xml:"putDataBySqhResult"`
}

func (s *Login) PutDataBySqh(req *PutDataBySqhReq) *PutDataBySqhResponse {
	fmt.Println("----------", req.Sqh, req.User)
	return &PutDataBySqhResponse{PutDataBySqhResult: "123"}
}

func (s *Login) Action() map[string]string {
	return map[string]string{
		"putDataBySqh": "PutDataBySqh",
	}
}
