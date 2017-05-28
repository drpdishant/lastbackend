//
// Last.Backend LLC CONFIDENTIAL
// __________________
//
// [2014] - [2017] Last.Backend LLC
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

package service

import (
	"fmt"
	n "github.com/lastbackend/lastbackend/pkg/api/namespace/views/v1"
	c "github.com/lastbackend/lastbackend/pkg/cli/context"
	"github.com/lastbackend/lastbackend/pkg/common/errors"
	"github.com/lastbackend/lastbackend/pkg/common/types"
	"time"
)

type createS struct {
	Namespace string  `json:"namespace,omitempty"`
	Name      string  `json:"name,omitempty"`
	Template  string  `json:"template,omitempty"`
	Image     string  `json:"image,omitempty"`
	Url       string  `json:"url,omitempty"`
	Config    *Config `json:"config,omitempty"`
}

type Config struct {
	Replicas int `json:"replicas,omitempty"`
	//Ports   []string `json:"ports,omitempty"`
	//EnvVars     []string `json:"env,omitempty"`
	//Volumes []string `json:"volumes,omitempty"`
}

func DeployCmd(name, image, template, url string, replicas int) {

	var (
		config  *Config

        // for spinner
		i       = 0
		spinner = []string{"/", "|", "\\", "|"}
	)

	if replicas != 0 /* || len(env) != 0 || len(ports) != 0 || len(volumes) != 0 */ {
		config = new(Config)
		config.Replicas = replicas
		//config.EnvVars = env
		//config.Ports = ports
		//config.Volumes = volumes
	}

	err := Deploy(name, image, template, url, config)
	if err != nil {
		fmt.Println(err)
		return
	}

	// spinner
	// waiting for start service
	for range time.Tick(time.Millisecond * 500) {
		srv, _, err := Inspect(name)
		if err != nil {
			fmt.Println(err)
			return
		}

		if srv.State.State == types.StateProvision {
			i++
			if i == 3 {
				i = 0
			}

			fmt.Printf("Waiting for start service %v\r", spinner[i])
			continue
		}

		break
	}

	fmt.Println("Service `" + name + "` is succesfully created")
}

func Deploy(name, image, template, url string, config *Config) error {

	var (
		err     error
		http    = c.Get().GetHttpClient()
		storage = c.Get().GetStorage()
		ns      = new(n.Namespace)
		er      = new(errors.Http)
		res     = new(struct{})
	)

	ns, err = storage.Namespace().Load()
	if err != nil {
		return err
	}

	if ns.Meta.Name == "" {
		return errors.New("Namespace didn't select")
	}

	var cfg = createS{}
	cfg.Namespace = ns.Meta.Name

	if name != "" {
		cfg.Name = name
	}

	if template != "" {
		cfg.Template = template
	}

	if image != "" {
		cfg.Image = image
	}

	if url != "" {
		cfg.Url = url
	}

	if config != nil {
		cfg.Config = config
	}

	_, _, err = http.
		POST(fmt.Sprintf("/namespace/%s/service", ns.Meta.Name)).
		AddHeader("Content-Type", "application/json").
		BodyJSON(cfg).
		Request(res, er)
	if err != nil {
		return errors.New(er.Message)
	}

	if er.Code == 401 {
		return errors.NotLoggedMessage
	}

	if er.Code != 0 {
		return errors.New(er.Message)
	}

	return nil
}
