package fnsnpd

import(
	"time"
	"strings"
	"errors"
	"unicode"
	"regexp"
	"strconv"
	"fmt"
	"net/http"
	"io"
	"os"
	
	"github.com/otiai10/gosseract/v2"	
)

// Работа с чеком физического лица-самозанятого
// Структура чека - CheckFl
// Методы получения данных:
//	NewCheckFlFromUrl() - данные чека по URL чека из ЛК МойНалог
//	NewCheckFlFromFile() - данные чека из файла png чека, скаченного с ЛК МойНалог

// Raised on text structure error
var ErUnknownStruct = errors.New("unknown structure")

// Raised when qr code does belong to FNS NPD
var ErQRNotNDP = errors.New("qr code is not npd")

const (
	NUMBER_LN = 0
	DATE_LN = 1
	NAME_LN = 2 //1 or 2 lines
	
	CHECK_DATE_TMPL = "02.01.06 15:04(-07:00)"
	TOTAL_MARK = "Итого: "
	TAX_TYPE_MARK = "режим НО "
	INN_MARK = "ИНН "
	BUYER_INN_MARK = "ИНН: "
	
	FNS_NPD_HOST = "https://lknpd.nalog.ru"
)

// CheckFlItem старуткура услуги чека самлзанятого.
type CheckFlItem struct {
	Name string `json:"name"`	//Наименование чистое, без нумерации
	Sum float32 `json:"sum"`	//Сумма
}

// CheckFl струтура чека самозанятого
type CheckFl struct {
	Num string `json:"num"`		//Номер чека
	Date time.Time
	Name string `json:"name"`	//ФИО, нормальное Иванов Иван Иванович
	Items []CheckFlItem
	Total float32 `json:"total"`	//Сумма
	TaxType string `json:"inn"`
	Inn string `json:"inn"`
	BuyerInn string `json:"byuerInn"`
}

// NewCheckFlFromUrl получает данные чека по URL чека из ЛК МойНалог.
func NewCheckFlFromUrl(requestURL string) (*CheckFl, error) {
	resp, err := http.Get(requestURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	file, err := os.CreateTemp("", "checkPrint")
	if err != nil {
		return nil, err
	}
	defer func(){
		file.Close()
		os.Remove(file.Name())
	}()
		
	if _, err = io.Copy(file, resp.Body); err != nil {
		return nil, err
	}

	return NewCheckFlFromFile(file.Name())
}

// ParseCheckImage парсит png файл чека самозанятого с сайта МойНалог.
// Возвращает структуру чека.
func NewCheckFlFromFile(imgFileName string) (*CheckFl, error) {
	client := gosseract.NewClient()
	defer client.Close()
	
	client.SetImage(imgFileName)
	client.SetLanguage("eng", "rus")
	text, err := client.Text()
	if err != nil {
		return nil, err
	}
//fmt.Println(text)	
	lines := strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n")
	
	//minimal length for header
	if len(lines) < NAME_LN + 1 + 1 {
		return nil, ErUnknownStruct
	}
	
	check := CheckFl{}
	
	//1) number
	num := strings.Split(lines[NUMBER_LN], " ")
	check.Num = num[len(num) - 1]
	if len(check.Num) > 2 && check.Num[0:2] == "Ne" {
		check.Num = check.Num[2:]
		
	}else if len(check.Num) > 2 && check.Num[0:1] == "№" {
		check.Num = check.Num[1:]
	}
	
	//2) date
	check.Date, err = time.Parse(CHECK_DATE_TMPL, lines[DATE_LN])
	if err != nil {
		return nil, err
	}
	
	parse_line := NAME_LN + 1
	
	//3) Name
	name_parts := strings.Split(lines[NAME_LN], " ")
	for i, s := range name_parts {
		if len(s) == 0 {
			continue
		}
		runes := []rune(strings.ToLower(s))
		runes[0] = unicode.ToUpper(runes[0])
		name_parts[i] = string(runes)
	}
	check_name := strings.Join(name_parts, " ")
	name_cont := lines[NAME_LN + 1]
	matched, _ := regexp.MatchString(`^[А-Я]+$`, name_cont) //next line if all russian letters uppercase
	if matched {
		//two lines
		parse_line++
		if len(name_cont) > 1 {
			runes := []rune(strings.ToLower(name_cont))
			runes[0] = unicode.ToUpper(runes[0])
			check_name += " " + string(runes)
		}else{
			check_name += " " + name_cont
		}
	}
	//check for only rus letters
	for _,lt := range []rune(check_name) {
		if (lt < 'А' || lt > 'я') && lt != ' ' {
			continue
		}
		check.Name+= string(lt)
	}
	
	parse_line++ //skeep item table header
	
	if len(lines) < parse_line + 1 {
		return nil, ErUnknownStruct
	}
	
	//items
	check.Items = make([]CheckFlItem, 0)
	item_i := 0
	item := CheckFlItem{}
	for {	
		if parse_line == len(lines) || strings.Contains(lines[parse_line], TOTAL_MARK){
			break
		}
		matched, _ := regexp.MatchString(fmt.Sprintf("^%d\\. .*$", item_i + 1), lines[parse_line])
		if !matched {
			//previous item
			if len(check.Items) > 0 {
				check.Items[len(check.Items) - 1].Name += " " + lines[parse_line]
			}
			
		}else{
			//new item
			pref := fmt.Sprintf("%d. ", item_i + 1) //remove prefix
			lines[parse_line] = lines[parse_line][len(pref):]
			
			item = CheckFlItem{}
			//price: backward string from spase to space
			item_parts := strings.Split(lines[parse_line], " ")
			if len(item_parts) < 2 {
				//no price ??
				item.Name = lines[parse_line]
			}else{
				//name + price + cur
				//TODO might be formatted string 1 000,00
				price_s := strings.Replace(item_parts[len(item_parts) - 2], ",", ".", 1)				
				if sm, err := strconv.ParseFloat(price_s, 32); err == nil {
					item.Sum = float32(sm)
				}
				item.Name = strings.Join(item_parts[0:len(item_parts) - 2], " ")
			}						
			check.Items = append(check.Items, item)
			item_i++
		}		
		parse_line++
	}
	
	tot_ind := strings.Index(lines[parse_line], TOTAL_MARK)
	if tot_ind < 0 {
		return nil, ErUnknownStruct
	}
	tot_ind+= len(TOTAL_MARK)		
	tot_parts := strings.Split(lines[parse_line][tot_ind:], " ")
	price_s := strings.Join(tot_parts[0:len(tot_parts)-1], "")
	price_s = strings.Replace(price_s, ",", ".", 1)
	if sm, err := strconv.ParseFloat(price_s, 32); err == nil {
		check.Total = float32(sm)
	}
	
	parse_line++
	for {
		if len(lines) == parse_line {
			break
		}
		
		if check.TaxType == "" {
			ind := strings.Index(lines[parse_line], TAX_TYPE_MARK)
			if ind >= 0 && len(lines[parse_line]) >len(TAX_TYPE_MARK) {
				check.TaxType = lines[parse_line][ind + len(TAX_TYPE_MARK):]
				continue
			}
		}
		if check.Inn == "" {
			ind := strings.Index(lines[parse_line], INN_MARK)
			if ind >= 0 && len(lines[parse_line]) >len(INN_MARK) {
				check.Inn = lines[parse_line][ind + len(INN_MARK):]
				continue
			}
		}
		if check.BuyerInn == "" {
			ind := strings.Index(lines[parse_line], BUYER_INN_MARK)
			if ind >= 0 && len(lines[parse_line]) >len(BUYER_INN_MARK) {
				check.BuyerInn = lines[parse_line][ind + len(BUYER_INN_MARK):]
				continue
			}
		}
		
		parse_line++
	}
	
	return &check, nil
}


