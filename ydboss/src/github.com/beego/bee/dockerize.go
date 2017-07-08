// Copyright 2016 bee authors
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	"flag"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"
)

var cmdDockerize = &Command{
	CustomFlags: true,
	UsageLine:   "dockerize",
	Short:       "Generates a Dockerfile for your Beego application",
	Long: `Dockerize generates a Dockerfile for your Beego Web Application.
  The Dockerfile will compile, get the dependencies with {{"godep"|bold}}, and set the entrypoint.

  {{"Example:"|bold}}
    $ bee dockerize -expose="3000,80,25"
  `,
	PreRun: func(cmd *Command, args []string) { ShowShortVersionBanner() },
	Run:    dockerizeApp,
}

const dockerBuildTemplate = `FROM {{.BaseImage}}

# Godep for vendoring
RUN go get github.com/tools/godep

# Recompile the standard library without CGO
RUN CGO_ENABLED=0 go install -a std

ENV APP_DIR $GOPATH{{.Appdir}}
RUN mkdir -p $APP_DIR

# Set the entrypoint
ENTRYPOINT $APP_DIR/{{.Entrypoint}}
ADD . $APP_DIR

# Compile the binary and statically link
RUN cd $APP_DIR
RUN CGO_ENABLED=0 godep go build -ldflags '-d -w -s'

EXPOSE {{.Expose}}
`

// Dockerfile holds the information about the Docker container.
type Dockerfile struct {
	BaseImage  string
	Appdir     string
	Entrypoint string
	Expose     string
}

var (
	expose    string
	baseImage string
)

func init() {
	fs := flag.NewFlagSet("dockerize", flag.ContinueOnError)
	fs.StringVar(&baseImage, "image", "library/golang", "Set the base image of the Docker container.")
	fs.StringVar(&expose, "expose", "8080", "Port(s) to expose in the Docker container.")
	cmdDockerize.Flag = *fs
}

func dockerizeApp(cmd *Command, args []string) int {
	if err := cmd.Flag.Parse(args); err != nil {
		logger.Fatalf("Error parsing flags: %v", err.Error())
	}

	logger.Info("Generating Dockerfile...")

	gopath := os.Getenv("GOPATH")
	dir, err := filepath.Abs(".")
	MustCheck(err)

	appdir := strings.Replace(dir, gopath, "", 1)

	// In case of multiple ports to expose inside the container,
	// replace all the commas with whitespaces.
	// See the verb EXPOSE in the Docker documentation.
	if strings.Contains(expose, ",") {
		expose = strings.Replace(expose, ",", " ", -1)
	}

	_, entrypoint := path.Split(appdir)
	dockerfile := Dockerfile{
		BaseImage:  baseImage,
		Appdir:     appdir,
		Entrypoint: entrypoint,
		Expose:     expose,
	}

	generateDockerfile(dockerfile)
	return 0
}

func generateDockerfile(df Dockerfile) {
	t := template.Must(template.New("dockerBuildTemplate").Parse(dockerBuildTemplate)).Funcs(BeeFuncMap())

	f, err := os.Create("Dockerfile")
	if err != nil {
		logger.Fatalf("Error writing Dockerfile: %v", err.Error())
	}
	defer CloseFile(f)

	t.Execute(f, df)

	logger.Success("Dockerfile generated.")
}
