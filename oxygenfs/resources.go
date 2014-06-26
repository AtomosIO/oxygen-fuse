package main

import (
	"bytes"
	"common"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

var _ = fmt.Print

type TitaniumClient struct {
	endpoint string
	email    string
	username string
	password string
	token    string
}

func NewTitaniumClient(endpoint string) *TitaniumClient {
	return &TitaniumClient{
		endpoint: endpoint,
	}
}

const (
	USERS_ENDPOINT    = "/users/"
	TOKENS_ENDPOINT   = "/tokens/"
	PROJECTS_ENDPOINT = "/projects/"
)

type CreateUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type CreateProjectRequest struct {
	ProjectName string `json:"project_name"`
	Token       string `json:"token"`
	Public      bool   `json:"public"`
}

type CreateTokenRequest struct {
	User     string `json:"user"`
	Password string `json:"password"`
}

type CreateTokenResponse struct {
	Token    string `json:"token,omitempty"`
	Username string `json:"username,omitempty"`
}

func (client *TitaniumClient) CreateToken() {
	// Create an invalid request with all incorrect fields

	request := CreateTokenRequest{
		User:     client.username,
		Password: client.password,
	}

	response := CreateTokenResponse{}
	_, respBody, _ := JSONPost(client.endpoint+TOKENS_ENDPOINT, request)
	json.Unmarshal(respBody, &response)

	client.token = response.Token
}

func (client *TitaniumClient) CreateRandomProject(public bool) string {
	// Create an invalid request with all incorrect fields
	projectName := common.RandomString(5)

	request := CreateProjectRequest{
		ProjectName: projectName,
		Token:       client.token,
		Public:      public,
	}

	JSONPost(client.endpoint+PROJECTS_ENDPOINT, request)
	return projectName
}

func (client *TitaniumClient) CreateRandomUser() {
	// Create an invalid request with all incorrect fields
	client.username = common.RandomUsername()
	client.email = common.RandomEmail()
	client.password = common.RandomPassword()

	request := CreateUserRequest{
		Username: client.username,
		Email:    client.email,
		Password: client.password,
	}

	JSONPost(client.endpoint+USERS_ENDPOINT, request)
	client.CreateToken()
}

func JSONPost(url string, jsonVar interface{}) (*http.Response, []byte, error) {
	// Marshal JSON structure
	body, err := json.Marshal(jsonVar)
	if err != nil {
		return nil, nil, err
	}

	// Send POST request
	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, nil, err
	}

	// Read body contents
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	resp.Body.Close()

	return resp, respBody, nil
}

func RandomByteSlice(length int) []byte {
	var bytes = make([]byte, length)
	//for i := 0; i < length; i++ {
	//		bytes[i] = byte(rand.Int())
	//	}
	rand.Read(bytes)
	return bytes
}

type TrackingReadCloser struct {
	readCloser io.ReadCloser
	offset     int64
}

func (trackingReadCloser *TrackingReadCloser) NewReader(readCloser io.ReadCloser, offset int64) {
	if trackingReadCloser.readCloser != nil {
		trackingReadCloser.readCloser.Close()
	}

	trackingReadCloser.readCloser = readCloser
	trackingReadCloser.offset = offset
}

func (trackingReadCloser *TrackingReadCloser) Read(buf []byte) (n int, err error) {
	n, err = trackingReadCloser.readCloser.Read(buf)
	trackingReadCloser.offset += int64(n)

	return n, err
}

func (trackingReadCloser *TrackingReadCloser) Close() (err error) {
	trackingReadCloser.offset = 0
	return trackingReadCloser.readCloser.Close()
}

type TrackingWriteCloser struct {
	writeCloser io.WriteCloser
	offset      int64
}

func (trackingWriteCloser *TrackingWriteCloser) NewWriter(writeCloser io.WriteCloser, offset int64) {
	if trackingWriteCloser.writeCloser != nil {
		trackingWriteCloser.writeCloser.Close()
	}

	trackingWriteCloser.writeCloser = writeCloser
	trackingWriteCloser.offset = offset
}

func (trackingWriteCloser *TrackingWriteCloser) Write(buf []byte) (n int, err error) {
	n, err = trackingWriteCloser.writeCloser.Write(buf)
	trackingWriteCloser.offset += int64(n)

	return n, err
}

func (trackingWriteCloser *TrackingWriteCloser) Close() (err error) {
	trackingWriteCloser.offset = 0
	return trackingWriteCloser.writeCloser.Close()
}

type PrintingReader struct {
	reader io.Reader
}

func (preader *PrintingReader) Read(p []byte) (int, error) {
	fmt.Printf("PrintingReader start read: %d\n", len(p))
	n, err := preader.reader.Read(p)
	fmt.Printf("PrintingReader read: %d, %s\n", n, err)
	return n, err
}

type ZeroReader struct{}

func NewZeroReader() *ZeroReader {
	return &ZeroReader{}
}

func (zeroReader *ZeroReader) Read(p []byte) (n int, err error) {
	for i := 0; i < len(p); i++ {
		p[i] = 0
	}
	return len(p), nil
}
