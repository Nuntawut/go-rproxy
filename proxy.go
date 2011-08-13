package main

import (
	"net"
	"json"
	"http"
	"os"
	"io"
	"log"
)

type Server map[string]string

func (s Server) ServeHTTP(out http.ResponseWriter, r *http.Request) {
	if addr, ok := s[r.Host]; ok {
		if c, e := net.Dial("tcp",addr); e == nil {
			if oc, _,e := out.(http.Hijacker).Hijack(); e == nil {
				go func() {
					io.Copy(oc, c)
					oc.Close()
				}()
				go func() {
					r.Header.Set("X-Forwarded-For",r.RemoteAddr)
					r.Write(c)
					io.Copy(c, oc)
					c.Close()
				}()
				if file,e := os.OpenFile("access.log", os.O_RDWR|os.O_APPEND|os.O_CREATE,0666); e == nil {
					log.SetOutput(io.MultiWriter(file,os.Stdout))
					log.Println(r.RemoteAddr,r.Host,r.Method,r.Header["Referer"],r.Proto,r.Header["User-Agent"])
					file.Close()
				}		
				return
			} else c.Close()
		} else os.Stderr.WriteString(os.Args[0] + ": " + e.String() + "\n")
	}
	log.Println("Service Unavailable")
	out.WriteHeader(503)
	out.Write([]byte("Service Unavailable"))
}

func main() {
	var (
		f io.Reader
		e os.Error
	)
	c := new(struct {
		Host string
		Services Server
	})
	if f, e = os.Open("config.json"); e == nil {
		if e = json.NewDecoder(f).Decode(c); e == nil {
			var l net.Listener
			if l, e = net.Listen("tcp", c.Host); e == nil {
				if e = http.Serve(l, c.Services); e == nil {
					os.Exit(0)
				}
			}
		}
	}
	os.Stderr.WriteString(os.Args[0] + ": " + e.String() + "\n")
	os.Exit(1)
}
