package shopapi

import (
	"CoffeeShop/coffeedb"
	"bytes"
	"encoding/json"
	"github.com/google/uuid"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func generateUserId(userAmount int) []string {
	usersId := make([]string, userAmount)
	for i := 0; i < userAmount; i++ {
		usersId[i] = uuid.New().String()
	}
	return usersId
}

func registerUserWithMembership(userId string, membership coffeedb.MembershipType, serverUrl string) (int, error) {
	postBody, _ := json.Marshal(UserRegister{
		UserId:     userId,
		Membership: membership,
	})
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(serverUrl+"/registerUser", "application/json", responseBody)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	//Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	sb := string(body)
	log.Printf(sb)

	return resp.StatusCode, err
}

func buyACoffeeForUser(userId string, cType coffeedb.CoffeeType, serverUrl string) (int, error) {
	postBody, _ := json.Marshal(CoffeeBuyInfo{
		UserId: userId,
		Coffee: cType,
	})
	responseBody := bytes.NewBuffer(postBody)
	resp, err := http.Post(serverUrl+"/buyCoffee", "application/json", responseBody)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, err
	}
	sb := string(body)
	log.Printf(sb)
	return resp.StatusCode, nil
}

func serverSetup() *httptest.Server {
	mux := http.NewServeMux()

	mux.Handle("/registerUser", http.HandlerFunc(apiRegisterUser))
	mux.Handle("/buyCoffee", http.HandlerFunc(apiBuyCoffee))

	return httptest.NewServer(mux)
}

func serverTeardown(s *httptest.Server) {
	s.Close()
}

func TestRegisterUsers(t *testing.T) {
	InitDefaultConfig()
	InitDb("Tmp")
	srv := serverSetup()

	testFailed := false
	usersId := generateUserId(100)
	for _, u := range usersId {
		respCode, err := registerUserWithMembership(u, coffeedb.Basic, srv.URL)
		if err != nil {
			log.Printf(err.Error())
		}
		if respCode != 200 {
			testFailed = true
			break
		}
	}
	serverTeardown(srv)

	clearDb()
	if testFailed {
		t.Fail()
	}
}

func TestBuyEspressoCoffee(t *testing.T) {
	customConfig := make(map[coffeedb.MembershipType]CoffeeQuotaPerMembership)

	//basic membership config
	basicEspressoCoffeeQuota := CoffeeQuota{Type: coffeedb.Espresso, Amount: 1, TimeFrame: int64(time.Second * 10)}
	basicAmericanoCoffeeQuota := CoffeeQuota{Type: coffeedb.Americano, Amount: 2, TimeFrame: int64(time.Second * 10)}
	basicCappuccinoCoffeeQuota := CoffeeQuota{Type: coffeedb.Cappuccino, Amount: 3, TimeFrame: int64(time.Second * 10)}
	basicQuotas := []CoffeeQuota{basicEspressoCoffeeQuota, basicAmericanoCoffeeQuota, basicCappuccinoCoffeeQuota}
	basicCoffeeConfig := CoffeeQuotaPerMembership{Membership: coffeedb.Basic, Quota: basicQuotas}
	customConfig[coffeedb.Basic] = basicCoffeeConfig

	InitWithConfig(customConfig)
	InitDb("Tmp")
	srv := serverSetup()

	testFailed := false
	resCode, err := registerUserWithMembership("e6b92500-6cbf-4848-ac51-1ff07c76d88e", coffeedb.Basic, srv.URL)
	if err != nil {
		log.Printf(err.Error())
	}
	if resCode != 200 {
		testFailed = true
	} else {
		//buy first time
		responseCode, err := buyACoffeeForUser("e6b92500-6cbf-4848-ac51-1ff07c76d88e", coffeedb.Espresso, srv.URL)
		if err != nil {
			t.Log(err)
			testFailed = true
		} else {
			if responseCode != http.StatusOK {
				testFailed = true
			}
		}
		time.Sleep(time.Second * 5)
		if !testFailed {
			//buy second time
			responseCode, err = buyACoffeeForUser("e6b92500-6cbf-4848-ac51-1ff07c76d88e", coffeedb.Espresso, srv.URL)
			if err != nil {
				t.Log(err)
				testFailed = true
			} else {
				if responseCode != http.StatusTooManyRequests {
					testFailed = true
				}
			}
		}
	}

	serverTeardown(srv)

	clearDb()
	if testFailed {
		t.Fail()
	}
}

func TestBuyEspressoCoffeeWithReset(t *testing.T) {
	customConfig := make(map[coffeedb.MembershipType]CoffeeQuotaPerMembership)

	//basic membership config
	basicEspressoCoffeeQuota := CoffeeQuota{Type: coffeedb.Espresso, Amount: 1, TimeFrame: int64(time.Second * 5)}
	basicAmericanoCoffeeQuota := CoffeeQuota{Type: coffeedb.Americano, Amount: 2, TimeFrame: int64(time.Second * 10)}
	basicCappuccinoCoffeeQuota := CoffeeQuota{Type: coffeedb.Cappuccino, Amount: 3, TimeFrame: int64(time.Second * 10)}
	basicQuotas := []CoffeeQuota{basicEspressoCoffeeQuota, basicAmericanoCoffeeQuota, basicCappuccinoCoffeeQuota}
	basicCoffeeConfig := CoffeeQuotaPerMembership{Membership: coffeedb.Basic, Quota: basicQuotas}
	customConfig[coffeedb.Basic] = basicCoffeeConfig

	InitWithConfig(customConfig)
	InitDb("Tmp")
	srv := serverSetup()

	testFailed := false
	resCode, err := registerUserWithMembership("e6b92500-6cbf-4848-ac51-1ff07c76d88e", coffeedb.Basic, srv.URL)
	if err != nil {
		log.Printf(err.Error())
	}
	if resCode != 200 {
		testFailed = true
	} else {
		//buy first time
		responseCode, err := buyACoffeeForUser("e6b92500-6cbf-4848-ac51-1ff07c76d88e", coffeedb.Espresso, srv.URL)
		if err != nil {
			t.Log(err)
			testFailed = true
		} else {
			if responseCode != http.StatusOK {
				testFailed = true
			}
		}
		time.Sleep(time.Second * 5)
		if !testFailed {
			//buy second time
			responseCode, err = buyACoffeeForUser("e6b92500-6cbf-4848-ac51-1ff07c76d88e", coffeedb.Espresso, srv.URL)
			if err != nil {
				t.Log(err)
				testFailed = true
			} else {
				if responseCode != http.StatusOK {
					testFailed = true
				}
			}
		}
	}

	serverTeardown(srv)

	clearDb()
	if testFailed {
		t.Fail()
	}
}
