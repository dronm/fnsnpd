package fnsnpd

import (
	"testing"
	"time"
	"os"
	"strconv"
	"sync"
)

const (
	VAR_TEST_CHECK_FILE = "TEST_CHECK_FILE"
	VAR_TEST_CHECK_IMG = "TEST_CHECK_IMG"		
	VAR_TEST_CHECK_NAME = "TEST_CHECK_NAME"
	VAR_TEST_CHECK_DATE = "TEST_CHECK_DATE"
	VAR_TEST_CHECK_NUM = "TEST_CHECK_NUM"
	VAR_TEST_CHECK_INN = "TEST_CHECK_INN"
	VAR_TEST_CHECK_BUYER_INN = "TEST_CHECK_BUYER_INN"
	VAR_TEST_CHECK_TOTAL = "TEST_CHECK_TOTAL"
	VAR_TEST_CHECK_ITEM0_NAME = "TEST_CHECK_ITEM0_NAME"
	VAR_TEST_CHECK_ITEM0_SUM = "TEST_CHECK_ITEM0_SUM"
	VAR_TEST_CHECK_URL = "TEST_CHECK_URL"
	
	INN_TEST_COUNT = 4
)

func getTestVar(t *testing.T, n string) string {
	v := os.Getenv(n)
	if v == "" {
		t.Fatalf("getTestVar() failed: '%s' environment variable is not set", n)
	}
	return v
}

func TestNewCheckFlFromFile(t *testing.T) {
	check, err := NewCheckFlFromFile(getTestVar(t, VAR_TEST_CHECK_FILE))
	if err != nil {
		t.Fatalf("NewCheckFlFromFile() failed: %v", err)
	}
	checkCheckData(t, check)
}

func TestInn(t *testing.T) {
	InitFNSPersonCheck(os.Stderr)
	
	var wg sync.WaitGroup
	for i := 0; i < INN_TEST_COUNT; i++ {
		wg.Add(1)
		go func(test_ind int){
			defer wg.Done()
			ch := PersonCheckerFNS.AddCheck(getTestVar(t, VAR_TEST_CHECK_INN))
			inn_fns_ok := <-ch //wait	
			if !inn_fns_ok {
				t.Fatal("Не найден по данным ФНС")
			}
			t.Logf("Тест: %d - OK", test_ind + 1)
		}(i)
	}
	
	wg.Wait()
}


func TestNewCheckFlFromUrl(t *testing.T) {
	check, err := NewCheckFlFromUrl(getTestVar(t, VAR_TEST_CHECK_URL))
	if err != nil {
		t.Fatalf("NewCheckFlFromUrl() failed: %v", err)
	}	
	checkCheckData(t, check)
}

// checkCheckData функция проверки полученного чека и исходных данных
func checkCheckData(t *testing.T, check *CheckFl) {	
	//name
	test_name := getTestVar(t, VAR_TEST_CHECK_NAME)
	if check.Name != test_name {
		t.Fatalf("Name expected to be: %s, got %s", test_name, check.Name)
	}
	
	//date
	test_date, err := time.Parse(CHECK_DATE_TMPL, getTestVar(t, VAR_TEST_CHECK_DATE))
	if err != nil {
		t.Fatalf("time.Parse() failed: %v", err)
	}
	if check.Date != test_date {
		t.Fatalf("Date expected to be: %s, got %s", test_date.Format(CHECK_DATE_TMPL), check.Date.Format(CHECK_DATE_TMPL))
	}
	
	//num
	/*test_num := getTestVar(t, VAR_TEST_CHECK_NUM)
	if check.Num != test_num {
		t.Fatalf("Number expected to be: %s, got %s", test_num, check.Num)
	}*/

	//Inn
	test_inn := getTestVar(t, VAR_TEST_CHECK_INN)
	if check.Inn != test_inn {
		t.Fatalf("INN expected to be: %s, got %s", test_inn, check.Inn)
	}

	//BuyerInn
	test_binn := getTestVar(t, VAR_TEST_CHECK_BUYER_INN)
	if check.BuyerInn != test_binn {
		t.Fatalf("INN expected to be: %s, got %s", test_binn, check.BuyerInn)
	}

	//Total
	test_total, err := strconv.ParseFloat(getTestVar(t, VAR_TEST_CHECK_TOTAL), 32)
	if err != nil {
		t.Fatalf("strconv.ParseFloat() failed: %v", err)
	}
	if check.Total != float32(test_total) {
		t.Fatalf("INN expected to be: %f, got %f", test_total, check.Total)
	}

	//Item0	
	if len(check.Items) == 0 {
		t.Fatal("len(Items) must be >0")
	}
	test_it0_name := getTestVar(t, VAR_TEST_CHECK_ITEM0_NAME)	
	if check.Items[0].Name != test_it0_name {
		t.Fatalf("Items[0].Name expected to be: %s, got %s", test_it0_name, check.Items[0].Name)
	}
	test_it0_price, err := strconv.ParseFloat(getTestVar(t, VAR_TEST_CHECK_ITEM0_SUM), 32)
	if err != nil {
		t.Fatalf("strconv.ParseFloat() failed: %v", err)
	}
	if check.Items[0].Sum != float32(test_it0_price) {
		t.Fatalf("Items[0].Sum expected to be: %f, got %f", test_it0_price, check.Items[0].Sum)
	}
	
}

