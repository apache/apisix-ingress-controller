package apisix

import (
	"strconv"
	"github.com/gxthrj/seven/apisix"
)

type CorsYaml struct {
	Enable       bool   `json:"enable,omitempty"`
	AllowOrigin  string `json:"allow_origin,omitempty"`
	AllowHeaders string `json:"allow_headers,omitempty"`
	AllowMethods string `json:"allow_methods,omitempty"`
}

func (c *CorsYaml) SetEnable(enable string){
	if b, err := strconv.ParseBool(enable); err != nil {
		c.Enable = false
	} else {
		c.Enable = b
	}
}

func (c *CorsYaml) SetOrigin(origin string){
	c.AllowOrigin = origin
}

func (c *CorsYaml) SetHeaders(headers string){
	c.AllowHeaders = headers
}
func (c *CorsYaml) SetMethods(methods string){
	c.AllowMethods = methods
}

func (c *CorsYaml) Build() *apisix.Cors{
	maxAge := int64(3600)
	return apisix.BuildCors(c.Enable, &c.AllowOrigin, &c.AllowHeaders, &c.AllowMethods, &maxAge)
}
