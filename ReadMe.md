# Проверка данных самозанятого
Реализованы методы для проверки ИНН самозанятых и данные чеков по сканам с [Мой налог](https://lknpd.nalog.ru)

### Как использовать:
- Для проверки ИНН самозанятого запускается сервер. Взаимодействие с сервером через функцию fnsnpd.PersonCheckerFNS.AddCheck(inn),
	которая возвращает канал bool с результатом проверки. Сервер требует инициализации функцией fnsnpd.InitFNSPersonCheck(os.Stderr).
	ФНС разрешает проверять чеки с периодичностью 2 проверки в минуту, поэтому после двух проверок сервер будет ожидать 
	время, оставшееся до двух минут со времени последней проверки. Таким образом проверки следует запускать в отдельных горутинах.
```go
	import (
		"os"
		"fmt"
		
		"github.com/dronm/fnsnpd"
	)

	//инициализация 
	fnsnpd.InitFNSPersonCheck(os.Stderr)
	
	inn := "<ИНН ДЛЯ ПРОВЕРКИ>"
	ch := fnsnpd.PersonCheckerFNS.AddCheck(inn)
	inn_fns_ok := <-ch //wait	
	if !inn_fns_ok {
		panic("Не найден по данным ФНС")
	}
	fmt.Println("ИНН существует по даннм ФНС")

```
	
- Для проверки чеков	
```go
	import (
		"fmt"
		
		"github.com/dronm/fnsnpd"
	)
	
	url := "https://lknpd.nalog.ru/api/v1/receipt/ИНН/НОМЕРЧЕКА/print"
	check, err := fnsnpd.NewCheckFromUrl(url)
	if err != nil {
		panic(fmt.Sprintf("fnsnpd.NewCheckFromUrl() failed: %v",err))
	}
	
	fmt.Printf("%+v\n", check)

```
- Для запуска тестирования установить переменные окружения: 	
	- export TEST_CHECK_IMG=print.png Имя файла с изображением чека для проверки.
	Используется для тестирования функции NewCheckFlFromFile().
	Следует использовать файлы с сайта [Мой налог](https://lknpd.nalog.ru).
	
	- export TEST_CHECK_NAME='Иванов Иван Иванович'
	- export TEST_CHECK_DATE='01.01.23 15:10(+02:00)'
	- export TEST_CHECK_NUM='11111'
	- export TEST_CHECK_INN=00000000000
	- export TEST_CHECK_BUYER_INN=0000000000
	- export TEST_CHECK_TOTAL=1000
	- export TEST_CHECK_ITEM0_NAME='УСЛУГА'
	- export TEST_CHECK_ITEM0_SUM=10000
	- export TEST_CHECK_URL='https://lknpd.nalog.ru/api/v1/receipt/ИНН/НОМЕРЧЕКА/print' 
- Запустить тест go test
