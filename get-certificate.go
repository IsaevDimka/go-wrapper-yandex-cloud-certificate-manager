package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hako/durafmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

func JSONPretty(Data interface{}) string {
	var out bytes.Buffer // buffer for pretty json
	jsonData, _ := json.Marshal(Data)
	jsonData = bytes.Replace(jsonData, []byte("\\u0026"), []byte("&"), -1)
	_ = json.Indent(&out, jsonData, "", "    ")
	return out.String()
}

func CreateDirIfNotExist(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		check(err)
	}
}

var (
	Info = Teal
	Warn = Yellow
	Fata = Red
)

var (
	Black   = Color("\033[1;30m%s\033[0m")
	Red     = Color("\033[1;31m%s\033[0m")
	Green   = Color("\033[1;32m%s\033[0m")
	Yellow  = Color("\033[1;33m%s\033[0m")
	Purple  = Color("\033[1;34m%s\033[0m")
	Magenta = Color("\033[1;35m%s\033[0m")
	Teal    = Color("\033[1;36m%s\033[0m")
	White   = Color("\033[1;37m%s\033[0m")
)

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

const certificateId = ""
const certificatesDir string = "./certificates/domain.com"

func main() {

	start := time.Now()

	fmt.Println(Info("Running..."))
	if _, err := os.Stat(certificatesDir); !os.IsNotExist(err) {
		// если папка существует

		// получаем iam токен
		iamToken := getYandexIamToken()

		// получаем инфу о сертификате
		getCertInfo(iamToken, certificateId)
		certInfo, err := ioutil.ReadFile(certificatesDir + "/certificates_info_" + certificateId + ".json")
		check(err)

		type Certificate struct {
			Domains          []string `json:"domains"`
			CertificateChain string   `json:"certificateChain"`
			FolderId         string   `json:"folderId"`
			CreatedAt        string   `json:"createdAt"`
			Name             string   `json:"name"`
			Type             string   `json:"type"`
			Status           string   `json:"status"`
			Issuer           string   `json:"issuer"`
			Subject          string   `json:"subject"`
			Serial           string   `json:"serial"`
			UpdatedAt        string   `json:"updatedAt"`
			IssuedAt         string   `json:"issuedAt"`
			NotAfter         string   `json:"notAfter"`
			NotBefore        string   `json:"notBefore"`
		}
		CertData := Certificate{}
		json.Unmarshal(certInfo, &CertData)

		time_now := time.Now()
		fmt.Printf("Now: %s\n", time_now.Format("2006-1-2"))

		time_expired_parse, err := time.Parse(time.RFC3339, CertData.NotAfter)
		check(err)
		fmt.Printf("Certificate expiration date: %s\n", time_expired_parse.Format("2006-1-2"))
		diff_days := time_expired_parse.Sub(time_now).Hours() / 24
		fmt.Println("Diff days: ", int8(diff_days))
		if diff_days < 7 {
			fmt.Println("⚠️ Certificate is valid until 7 days")
			yandex_certificates(iamToken, certificateId)
		} else {
			fmt.Println("👍  Okay")
		}
	} else {
		CreateDirIfNotExist(certificatesDir)
		// set & get iam token
		iamToken := getYandexIamToken()
		getCertInfo(iamToken, certificateId)
		// download & formatting certificate
		yandex_certificates(iamToken, certificateId)
	}

	timeduration := time.Since(start)
	duration := durafmt.Parse(timeduration).LimitFirstN(2).String()
	fmt.Println(duration)
}

func getCertInfo(iamToken string, certificateId string) {
	fmt.Println("Get cert info for certificateId: " + certificateId)
	url := "https://certificate-manager.api.cloud.yandex.net/certificate-manager/v1/certificates/" + certificateId

	req, err := http.NewRequest("GET", url, nil)

	req.Header.Set("Authorization", "Bearer "+iamToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	check(err)
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)

	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, body, "", "\t")
	if error != nil {
		log.Println("JSON parse error: ", error)
		return
	}
	// save response
	createAndWriteInFile(certificatesDir+"/certificates_info_"+certificateId+".json", string(prettyJSON.Bytes()))
}

func getYandexIamToken() string {
	fmt.Println("Get yandex iamToken")

	// get oAuth token https://cloud.yandex.ru/docs/iam/operations/iam-token/create
	reqBody := bytes.NewBuffer([]byte(`{"yandexPassportOauthToken":"YOUR-TOKEN"}`))
	req, err := http.NewRequest("POST", "https://iam.api.cloud.yandex.net/iam/v1/tokens", reqBody)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	check(err)
	defer resp.Body.Close()

	//fmt.Println("response Status:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)

	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, body, "", "\t")
	if error != nil {
		log.Println("JSON parse error: ", error)
		return ""
	}

	// write in file
	createAndWriteInFile(certificatesDir+"/iam_token.json", string(prettyJSON.Bytes()))

	var jsonStr = []byte(string(prettyJSON.Bytes()))
	//fmt.Println("Response", string(jsonStr))

	type Json struct {
		IamToken  string `json:"iamToken"`
		ExpiresAt string `json:"expiresAt"`
	}
	data := Json{}
	json.Unmarshal(jsonStr, &data)

	fmt.Println("iamToken: " + data.IamToken)
	fmt.Println("expiresAt: " + data.ExpiresAt)

	return data.IamToken
}

func yandex_certificates(iamToken string, certificateId string) {

	url := "https://data.certificate-manager.api.cloud.yandex.net/certificate-manager/v1/certificates/" + certificateId + ":getContent"

	req, err := http.NewRequest("GET", url, nil)

	req.Header.Set("Authorization", "Bearer "+iamToken)

	client := &http.Client{}
	resp, err := client.Do(req)
	check(err)
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	body, _ := ioutil.ReadAll(resp.Body)

	var prettyJSON bytes.Buffer
	error := json.Indent(&prettyJSON, body, "", "\t")
	if error != nil {
		log.Println("JSON parse error: ", error)
		return
	}

	// save response
	createAndWriteInFile(certificatesDir+"/certificates.json", string(prettyJSON.Bytes()))

	var jsonStr = []byte(string(prettyJSON.Bytes()))

	//fmt.Println("jsonStr", string(jsonStr))
	type Json struct {
		CertificateId    string   `json:"certificateId"`
		CertificateChain []string `json:"certificateChain"`
		PrivateKey       string   `json:"privateKey"`
	}
	data := Json{}
	json.Unmarshal(jsonStr, &data)

	fmt.Println("CertificateId: ", data.CertificateId)

	createAndWriteInFile(certificatesDir+"/cert.pem", data.CertificateChain[0])
	createAndWriteInFile(certificatesDir+"/chain.pem", data.CertificateChain[1])
	createAndWriteInFile(certificatesDir+"/fullchain.pem", data.CertificateChain[0]+data.CertificateChain[1])
	createAndWriteInFile(certificatesDir+"/privkey.pem", data.PrivateKey)
}

func createAndWriteInFile(path string, text string) {
	fmt.Println("Create and write data in file: " + path)
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err2 := f.WriteString(text)
	if err2 != nil {
		log.Fatal(err2)
	}
}
