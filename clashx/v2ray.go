package clashx

import "encoding/json"

type V2rayConfig struct {
	Add  string      `json:"add"`
	Host string      `json:"host"`
	Id   string      `json:"id"`
	Net  string      `json:"net"`
	Path string      `json:"path"`
	Port json.Number `json:"port"`
	Ps   string      `json:"ps"`
	TLS  string      `json:"tls"`
	V    int         `json:"v"`
	Aid  int         `json:"aid"`
	Type string      `json:"type"`
}
