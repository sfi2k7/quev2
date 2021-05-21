package quev2

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
)

var (
	baseAddress = "http://localhost:7676/"
)

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

func (qc *QueClient) Move(froml, tolist, item string) error {
	finalURL := fmt.Sprintf("%s?%s", baseAddress, "fl="+froml+"&tl="+tolist+"&i="+item)

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
