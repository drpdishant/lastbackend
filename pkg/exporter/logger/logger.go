//
// Last.Backend LLC CONFIDENTIAL
// __________________
//
// [2014] - [2018] Last.Backend LLC
// All Rights Reserved.
//
// NOTICE:  All information contained herein is, and remains
// the property of Last.Backend LLC and its suppliers,
// if any.  The intellectual and technical concepts contained
// herein are proprietary to Last.Backend LLC
// and its suppliers and may be covered by Russian Federation and Foreign Patents,
// patents in process, and are protected by trade secret or copyright law.
// Dissemination of this information or reproduction of this material
// is strictly forbidden unless prior written permission is obtained
// from Last.Backend LLC.
//

package logger

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/lastbackend/lastbackend/pkg/distribution/types"
	"github.com/lastbackend/lastbackend/pkg/log"
	"github.com/spf13/viper"

	"net/http"

	"github.com/lastbackend/lastbackend/pkg/util/proxy"
)

const (
	logPrefix = "exporter:logger"
	logLevel  = 3
)

type Logger struct {
	server      *proxy.Server
	connections map[string]map[http.ResponseWriter]bool
	storage     *Storage
}

type StreamOpts struct {
	Follow bool
	Lines  int
}

func (l *Logger) Listen() error {
	return l.server.Listen(l.Handle)
}

func (l *Logger) Handle(msg types.ProxyMessage) error {

	var (
		stream *File
		err    error
	)

	m := types.LogMessage{}
	if err := json.Unmarshal(msg.Line, &m); err != nil {
		_ = fmt.Errorf("%s:>unmarshal json: %s", logPrefix, err.Error())
		return nil
	}

	pod := types.PodSelfLink{}
	if err := pod.Parse(m.Selflink); err != nil {
		return nil
	}

	k, parent := pod.Parent()

	switch k {
	case types.KindDeployment:
		kind, p := parent.Parent()
		stream, err = l.storage.GetStream(kind, p.String(), false)
	case types.KindTask:
		stream, err = l.storage.GetStream(k, parent.String(), false)
	}
	if err != nil {
		log.Errorf("get stream err: %s", err.Error())
		return err
	}

	if _, err := stream.Write(msg.Line); err != nil {
		return nil
	}

	return nil
}

func (l *Logger) Stream(ctx context.Context, kind, selflink string, opts StreamOpts, writer http.ResponseWriter) error {

	var (
		done = make(chan bool)
		lch  = make(chan string)
	)

	defer func() {
		close(done)
		close(lch)
	}()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				return
			case l := <-lch:

				if writer == nil {
					return
				}

				if _, err := writer.Write([]byte(l)); err != nil {
					return
				}

				if f, ok := writer.(http.Flusher); ok {
					if writer != nil {
						f.Flush()
					}
				}
			}
		}
	}()

	f, err := l.storage.GetStream(kind, selflink, false)
	if err != nil {
		log.Errorf("%s:> get stream err: %s", logPrefix, err.Error())
		return err
	}

	if err := f.Read(ctx, opts.Lines, opts.Follow, lch); err != nil {
		log.Errorf("%s:> read stream err: %s", logPrefix, err.Error())
		return err
	}
	return nil

}

func NewLogger() (*Logger, error) {

	var (
		err  error
		root = viper.GetString("exporter.dir")
		host = viper.GetString("exporter.listener.host")
		port = viper.GetInt("exporter.listener.port")
	)

	l := new(Logger)
	l.connections = make(map[string]map[http.ResponseWriter]bool, 0)
	l.server, err = proxy.NewServer(fmt.Sprintf("%s:%d", host, port))
	l.storage = NewStorage(root)
	if err != nil {
		return nil, err
	}

	return l, nil
}
