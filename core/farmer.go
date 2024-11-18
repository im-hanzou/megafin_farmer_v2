package core

import (
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/valyala/fasthttp"
	"log"
	"megafin_farmer/customTypes"
	"strings"
	"time"
)

func doRequest(client *fasthttp.Client,
	url string,
	method string,
	payload interface{},
	headers map[string]string) ([]byte, error) {

	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.Header.SetMethod(strings.ToUpper(method))
	req.SetRequestURI(url)
	req.Header.SetContentType("application/json")

	if payload != nil {
		jsonData, err := json.Marshal(payload)

		if err != nil {
			return nil, fmt.Errorf("failed to marshal JSON: %w", err)
		}
		req.SetBody(jsonData)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := client.Do(req, resp); err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	respBody := make([]byte, len(resp.Body()))
	copy(respBody, resp.Body())
	return respBody, nil
}

func profileRequest(client *fasthttp.Client,
	privateKeyHex string,
	headers map[string]string) {
	for {
		var responseData customTypes.ProfileResponseStruct

		respBody, err := doRequest(client, "https://api.megafin.xyz/users/profile", "GET", nil, headers)

		if err != nil {
			log.Printf("%s | Error When Profile: %s", privateKeyHex, err)
			continue
		}

		if strings.Contains(string(respBody), "title>Access denied | api.megafin.xyz used Cloudflare to restrict access</title>") {
			log.Printf("%s | CloudFlare", privateKeyHex)
			continue
		}

		if err = json.Unmarshal(respBody, &responseData); err != nil {
			log.Printf("%s | Failed To Parse JSON Response When Profile: %s", privateKeyHex, string(respBody))
			continue
		}

		return
	}
}

func loginAccount(client *fasthttp.Client,
	privateKeyHex string,
	headers map[string]string) string {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)

	if err != nil {
		log.Panicf("%s | Failed to parse private key: %v", privateKeyHex, err)
	}

	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	signText := fmt.Sprintf("megafin.xyz requests you to sign in with your wallet address: %s", address.Hex())
	data := accounts.TextHash([]byte(signText))
	signature, err := crypto.Sign(data, privateKey)

	if err != nil {
		log.Panicf("%s | Failed to sign message: %v", privateKeyHex, err)
	}

	signature[64] += 27
	signHash := fmt.Sprintf("0x%x", signature)

	payload := map[string]interface{}{
		"invite_code": "133d76e4",
		"key":         address.String(),
		"wallet_hash": signHash,
	}

	for {
		var responseData customTypes.LoginResponseStruct
		respBody, err := doRequest(client, "https://api.megafin.xyz/auth", "POST", payload, headers)

		if err != nil {
			log.Printf("%s | Error When Auth: %s", privateKeyHex, err)
			continue
		}

		if strings.Contains(string(respBody), "title>Access denied | api.megafin.xyz used Cloudflare to restrict access</title>") {
			log.Printf("%s | CloudFlare", privateKeyHex)
			continue
		}

		if err = json.Unmarshal(respBody, &responseData); err != nil {
			log.Printf("%s | Failed To Parse JSON Response When Logging: %s", privateKeyHex, string(respBody))
			continue
		}

		return responseData.Result.Token
	}
}

func sendConnectRequest(client *fasthttp.Client,
	privateKeyHex string,
	headers map[string]string) (float64, float64) {
	for {
		var responseData customTypes.PingResponseStruct

		respBody, err := doRequest(client, "https://api.megafin.xyz/users/connect", "GET", nil, headers)

		if err != nil {
			log.Printf("%s | Error When Pinging: %s", privateKeyHex, err)
			continue
		}

		if strings.Contains(string(respBody), "title>Access denied | api.megafin.xyz used Cloudflare to restrict access</title>") {
			log.Printf("%s | CloudFlare", privateKeyHex)
			continue
		}

		if err = json.Unmarshal(respBody, &responseData); err != nil {
			log.Printf("%s | Failed To Parse JSON Response When Pinging: %s", privateKeyHex, string(respBody))
			continue
		}

		return responseData.Result.Balance.MGF, responseData.Result.Balance.USDC
	}
}

func StartFarmAccount(privateKey string,
	proxy string) {
	headers := map[string]string{
		"accept":          "*/*",
		"accept-language": "ru,en;q=0.9,vi;q=0.8,es;q=0.7,cy;q=0.6",
		"origin":          "https://app.megafin.xyz",
		"referer":         "https://app.megafin.xyz",
		"connection":      "close",
	}

	client := GetClient(proxy)
	authToken := loginAccount(client, privateKey, headers)
	headers["Authorization"] = "Bearer " + authToken
	profileRequest(client, privateKey, headers)

	for {
		mgfBalance, usdcBalance := sendConnectRequest(client, privateKey, headers)
		log.Printf("%s | MGF Balance: %f | USDC Balance: %f | Sleeping 90 secs.", privateKey, mgfBalance, usdcBalance)
		time.Sleep(time.Second * time.Duration(90))
	}
}

func ParseAccountBalance(privateKey string,
	proxy string) (float64, float64) {
	headers := map[string]string{
		"accept":          "*/*",
		"accept-language": "ru,en;q=0.9,vi;q=0.8,es;q=0.7,cy;q=0.6",
		"origin":          "https://app.megafin.xyz",
		"referer":         "https://app.megafin.xyz",
		"connection":      "close",
	}

	client := GetClient(proxy)
	authToken := loginAccount(client, privateKey, headers)
	headers["Authorization"] = "Bearer " + authToken
	profileRequest(client, privateKey, headers)
	mgfBalance, usdcBalance := sendConnectRequest(client, privateKey, headers)

	log.Printf("%s | MGF Balance: %f | USDC Balance: %f", privateKey, mgfBalance, usdcBalance)

	return mgfBalance, usdcBalance
}
