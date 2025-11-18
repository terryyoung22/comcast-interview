# Comcast-interview

## Commands to build image locally / Build Docker image

**For all these builds locally with cli or docker, please make sure you have a local server running. Feel free to run the command in the `server.txt` file. `wrongserver.txt` will give you the wrong langues error `{"detected":"en-MX","expected":"en-US","explanation":"language mismatch","type":"incorrect_language"}`. Can change in the text which since we arent hitting public enpoint.
**Feel free to use other files types as well

### Local
**This will be so you can test local with no build:
- `cd app`
- `go mod init app`
- `go mod tidy`
- If ok, run the command from the server text to start a local server then run commands from below from app
- `go run . -t_start 0 -t_end 10 -min_coverage_pct 90 -endpoint http://localhost:8080 ../samples/work.vtt` #This should give you a success with current configurations
- `go run . -t_start 0 -t_end 10 -min_coverage_pct 90 -endpoint http://localhost:8080 ../samples/test.vtt` #This should give you a failure
- `go run . -t_start 0 -t_end 10 -min_coverage_pct 90 -endpoint http://localhost:8080 ../samples/test.srt` #same here
- `go run . -t_start 0 -t_end 10 -min_coverage_pct 90 -endpoint http://localhost:8080 ../samples/work.svv` # exit status 1

**This is so you can build the package and test:
- `cd app`
- If you are working off init app above, you can use `go build -o app`
- Now run:
- `./app -t_start 0 -t_end 10 -min_coverage_pct 90 -endpoint http://localhost:8080 ../samples/work.vtt` #This should give you a success with current configurations
- `./app -t_start 0 -t_end 10 -min_coverage_pct 90 -endpoint http://localhost:8080 ../samples/test.vtt` #This should give you a failure
- `./app -t_start 0 -t_end 10 -min_coverage_pct 90 -endpoint http://localhost:8080 ../samples/test.srt` #same here

### DockerBuild and Run
- in the directory of the docker file. run `docker build -t comcast .`
- Where ever the docker image is stored, Make sure you are signed in to the repo
- From there you can test and run: 
- `docker run --rm -v "$(pwd)/samples":/app/samples comcast -t_start 0 -t_end 10 -min_coverage_pct 90 -endpoint http://host.docker.internal:8080  ./samples/test.vtt`
- `docker run --rm -v "$(pwd)/samples":/app/samples comcast -t_start 0 -t_end 10 -min_coverage_pct 90 -endpoint http://host.docker.internal:8080  ./samples/work.vtt`
