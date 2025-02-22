package parser

import (
	"errors"
	"io"
	"os/exec"
)

func Exec(name string, args ...string) ([]byte, error) {
	cmdName, err := exec.LookPath(name) // cmdName is absolute path
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(cmdName, args...)
	return getResult(cmd)
}

func getCmdReader(cmd *exec.Cmd) (io.ReadCloser, io.ReadCloser, error) {
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	err = cmd.Start()
	if err != nil {
		return nil, nil, err
	}

	return stdout, stderr, nil
}

func getResult(cmd *exec.Cmd) ([]byte, error) {
	stdout, stderr, err := getCmdReader(cmd)
	if err != nil {
		return nil, err
	}

	bytes, err := io.ReadAll(stdout)
	if err != nil {
		return nil, err
	}

	bytesErr, err := io.ReadAll(stderr)
	if err != nil {
		return nil, err
	}

	err = cmd.Wait()
	if err != nil {
		if len(bytesErr) != 0 {
			return nil, errors.New(string(bytesErr))
		}
		return nil, err
	}

	return bytes, nil
}
