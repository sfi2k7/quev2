package quev2

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
)

var (
	baseAddress = "http://52.1.190.104:7676/"
)

func init() {
	// 	fileName := "/var/que2pauser/pauser.data"
	// 	_, err := os.Stat(fileName)
	// 	fmt.Println("Creating Pauser", err)
	// 	if err != nil {
	// 		data := `1 N
	// 2 N
	// 3 N`
	// 		ioutil.WriteFile(fileName, []byte(data), 0777)
	// 	}
}

type QueClient struct {
}

func (qc *QueClient) Get(listname string) ([]string, error) {

	finalURL := fmt.Sprintf("%s?%s", baseAddress, "l="+listname)

	req, err := http.NewRequest("GET", finalURL, nil)
	if err != nil {
		return nil, err
	}
	c := http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	bts, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var one ApiResponse
	err = json.Unmarshal(bts, &one)
	if err != nil {
		return nil, err
	}

	if one.Success == false {
		return nil, errors.New(one.Error)
	}

	return one.Result, nil
}

func (qc *QueClient) Set(listname, item string) error {
	finalURL := fmt.Sprintf("%s?%s", baseAddress, "l="+listname+"&i="+item)

	req, err := http.NewRequest("POST", finalURL, nil)
	if err != nil {
		return err
	}
	c := http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bts, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var one ApiResponse
	err = json.Unmarshal(bts, &one)
	if err != nil {
		return err
	}

	if one.Success == false {
		return errors.New(one.Error)
	}

	return nil
}

func (qc *QueClient) Remove(listname, item string) error {
	finalURL := fmt.Sprintf("%s?%s", baseAddress, "l="+listname+"&i="+item)

	req, err := http.NewRequest("DELETE", finalURL, nil)
	if err != nil {
		return err
	}
	c := http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bts, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var one ApiResponse
	err = json.Unmarshal(bts, &one)
	if err != nil {
		return err
	}

	if one.Success == false {
		return errors.New(one.Error)
	}

	return nil
}

func (qc *QueClient) Move(tolist, item string) error {
	finalURL := fmt.Sprintf("%s?%s", baseAddress, "tl="+tolist+"&i="+item)

	req, err := http.NewRequest("PUT", finalURL, nil)
	if err != nil {
		return err
	}
	c := http.Client{}
	res, err := c.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	bts, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	var one ApiResponse
	err = json.Unmarshal(bts, &one)
	if err != nil {
		return err
	}

	if one.Success == false {
		return errors.New(one.Error)
	}

	return nil
}

func (qc *QueClient) IsComponentPaused(c int) bool {

	fileName := "/var/que2pauser/pauser.data"
	f, err := os.Open(fileName)
	if err != nil {
		return true
	}
	defer f.Close()

	cid := strconv.Itoa(c)
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		splitted := strings.Split(line, " ")
		if splitted[0] == cid && splitted[1] == "Y" {
			return true
		}
	}
	return false
}
