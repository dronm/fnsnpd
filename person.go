package fnsnpd

import(
	"fmt"
	"time"
	"net/http"
	"encoding/json"
	"bytes"
	"io"
)

//Проверка самозанятого. Согласно описанию ФНС можно
//отправлять 2 запроса в минуту

const (
	FNS_PERSON_CHECK_URL = "https://statusnpd.nalog.ru:443/api/v1/tracker/taxpayer_status"
	FNS_ALLOWED_PERIOD_SEC = 120
	FNS_ALLOWED_PERIOD_CNT = 2
	
	ERR_QUERY = "ошибка при выполнении запроса к ресурсу ФНС"
)

var PersonCheckerFNS *fnsCheckPerson

type FNSResponse struct {
	Code string `json:"code"`
	Status bool `json:"status"`
	Message string `json:"message"`
}

type PersonData struct {
	INN string
	CheckResult chan bool
}

type fnsCheckPerson struct {
	errLogger io.Writer
	personData chan *PersonData
	lastCheck time.Time
	checkCount int	
}

// AddCheck Добавление ИНН для проверки
func (c *fnsCheckPerson) AddCheck(inn string) chan bool {
	res := make(chan bool)
	c.personData<- &PersonData{INN: inn, CheckResult: res}
	return res
}

// Собственно поверка ИНН
func (c *fnsCheckPerson) check(p *PersonData) {
	check_res := false
	defer (func(){
		p.CheckResult<- check_res	
	})()
	
	json_data := []byte(fmt.Sprintf(`{"inn": "%s","requestDate": "%s"}`, p.INN, time.Now().Format("2006-01-02")))	
	req, _ := http.NewRequest("POST", FNS_PERSON_CHECK_URL, bytes.NewBuffer(json_data))	
	req.Header.Set("Content-Type", "application/json")			
	client := &http.Client{}
	resp, err := client.Do(req)	
	if err != nil {
		if c.errLogger != nil {
			c.errLogger.Write([]byte(fmt.Sprintf("error %s http.Do() failed: %v\n", time.Now().Format(time.RFC3339), err)))
		}
		return
	}
	defer resp.Body.Close()	
	
	c.lastCheck = time.Now()
	c.checkCount++
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.errLogger != nil {
			c.errLogger.Write([]byte(fmt.Sprintf("error %s io.ReadAll() failed: %v\n", time.Now().Format(time.RFC3339), err)))
		}
		return
	}
	fns_resp := FNSResponse{}
	if err := json.Unmarshal(body, &fns_resp); err != nil {
		if c.errLogger != nil {
			c.errLogger.Write([]byte(fmt.Sprintf("error %s json.Unmarsha() io.ReadAll() failed: %v\n", time.Now().Format(time.RFC3339), err)))
		}
		return
	}
	//fns_resp := FNSResponse{Status: true, Message: "Самозянятый существует"}
	
	
	if !fns_resp.Status {
		msg := ""
		if fns_resp.Message != "" {
			msg = fns_resp.Message
			
		}else if fns_resp.Code != "" {
			msg = fns_resp.Code
		//}else{
		//	msg = fmt.Sprintf("http %d", resp.StatusCode)
		}
		if c.errLogger != nil {
			c.errLogger.Write([]byte(fmt.Sprintf("error %s http request failed: %s\n", time.Now().Format(time.RFC3339), msg)))
		}
	}else{
		check_res = true
	}
}

func InitFNSPersonCheck(errLogger io.Writer) {
	PersonCheckerFNS = &fnsCheckPerson{errLogger: errLogger,
		personData: make(chan *PersonData),
	}
	PersonCheckerFNS.lastCheck = time.Now()
	go func(){		
		for p := range PersonCheckerFNS.personData {			
			diff := time.Now().Sub(PersonCheckerFNS.lastCheck)
			if diff.Seconds() <= FNS_ALLOWED_PERIOD_SEC && PersonCheckerFNS.checkCount < FNS_ALLOWED_PERIOD_CNT {
				PersonCheckerFNS.check(p)
			}else{
				w := time.Duration(FNS_ALLOWED_PERIOD_SEC * time.Second) - diff
				time.Sleep(w)
				PersonCheckerFNS.checkCount = 0
				PersonCheckerFNS.check(p)
			}
		}
	}()
}
