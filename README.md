# simple-http-multiplexer (тестовое задание от НПО Фарватер)

## Содержание:
- [Постановка задачи](#Task)
- [Описание проделанной работы по задачам](#Done)
- [Краткое описание особенностей реализации](#Desc)
- [Сборка и конфигурация](#Build)
- [Перспективы развития](#Perspective)

## <a name="Task"></a> Постановка задачи
1.  Приложение представляет собой http-сервер с одним хендлером   
2. Хендлер на вход получает POST-запрос со списком url в json-формате, сервер запрашивает данные по всем этим url и возвращает результат клиенту в json-формате
3. Если в процессе обработки хотя бы одного из url получена ошибка, обработка всего списка прекращается и клиенту возвращается текстовая ошибка
4. Для реализации задачи следует использовать Go 1.13 или выше
5. Использовать можно только компоненты стандартной библиотеки Go
6. Сервер не принимает запрос если количество url в в нем больше 20
7. Сервер не обслуживает больше чем 100 одновременных входящих http-запросов
8. Таймаут на обработку одного входящего запроса - 10 секунд
9. Для каждого входящего запроса должно быть не больше 4 одновременных исходящих
10. Таймаут на запрос одного url - секунда
11. Обработка запроса может быть отменена клиентом в любой момент, это должно повлечь за собой остановку всех операций связанных с этим запросом
12. Сервис должен поддерживать 'graceful shutdown': при получении сигнала от OS перестать принимать входящие запросы, завершить текущие запросы и остановиться
13. Результат должен быть выложен на github и запускаться docker-compose

## <a name="Done"></a> Описание проделанной работы по задачам
1. ✅ Привязка хендлера в [cmd/simple-http-multiplexer/main.go](https://github.com/Ghytro/simple-http-multiplexer/blob/main/cmd/simple-http-multiplexer/main.go)
2. ✅ Реализация хендлера в [internal/handler/multiplexer.go](https://github.com/Ghytro/simple-http-multiplexer/blob/main/internal/handler/multiplexer.go)
3. ✅ В файле с реализацией хендлера из пункта 2 предусмотрено несколько типов ошибок с различными HTTP-кодами ответа. Неизвестные ошибки возвращаются с кодом 500, таймауты возвращаются с кодом 408. При некорректном формате запроса возвращается ответ с кодом 400. Также предусмотрен возврат кода 429, когда достигнут лимит одновременных подключений к серверу (подробнее об этом в пункте 7).
4. ✅ Сервис реализован на Go 1.18 ([go.mod](https://github.com/Ghytro/simple-http-multiplexer/blob/main/go.mod))
5. ✅ Были использованы только компоненты стандартной библиотеки (go.sum отсутствует)
6. ✅ Обработка данной ошибки есть в хендлере в [internal/handler/multiplexer.go:160](https://github.com/Ghytro/simple-http-multiplexer/blob/main/internal/handler/multiplexer.go#L160)
7. ✅ Лимитер по количеству одновременных входящих подключений реализован в [internal/handler/muxwrappers.go:10](https://github.com/Ghytro/simple-http-multiplexer/blob/main/internal/handler/muxwrappers.go#L10)
8. ✅ Таймауты есть в хендлере в [internal/handler/multiplexer.go:201](https://github.com/Ghytro/simple-http-multiplexer/blob/main/internal/handler/multiplexer.go#L201)
9. ✅ Для каждого входящего запроса параллельно запускается не более 4 исходящих [internal/handler/multiplexer.go:126](https://github.com/Ghytro/simple-http-multiplexer/blob/main/internal/handler/multiplexer.go#L126)
10. ✅ Таймаут одного запроса - секунда [internal/handler/multiplexer.go:101](https://github.com/Ghytro/simple-http-multiplexer/blob/main/internal/handler/multiplexer.go#L101)
11. ✅ Обработка запроса может быть отменена клиентом в любой момент [internal/handler/multiplexer.go:207](https://github.com/Ghytro/simple-http-multiplexer/blob/main/internal/handler/multiplexer.go#L207) (помимо обработки контекста запроса в указанном блоке select, есть его обработка и при выполнении запросов по заданным url ([см. здесь](https://github.com/Ghytro/simple-http-multiplexer/blob/main/internal/handler/multiplexer.go#L89)))
12. ✅ Сервис поддерживает graceful shutdown: [internal/server/httpserver.go](https://github.com/Ghytro/simple-http-multiplexer/blob/main/internal/server/httpserver.go)
13. ✅ Подробнее о сборке и запуске в Docker-Compose в разделе "[Сборка и конфигурация](#Build)"

Помимо основных задач, дополнительно была реализована тестирующая функция для проверки работы хендлера: [test/handler_test.go](https://github.com/Ghytro/simple-http-multiplexer/blob/main/test/handler_test.go). Был учтен запуск тестов при сборке Docker-образа (они запускаются при выборе нужной цели сборки). Подробнее об этом будет рассказано в разделе "[Сборка и конфигурация](#Build)". Были учтены дополнительные ошибки, об обработке которых не было уточнено в постановке задачи. Подробнее об этом в разделе "[Краткое описание особенностей реализации](#Desc)"

## <a name="Desc"></a> Краткое описание особенностей реализации

Сервис принимает запрос в формате json, причем заголовок ```Content-Type: application/json``` может быть не указан клиентом. Если сервис получает данные не в формате json, возвращается ошибочный ответ с кодом 400, с соответствующим сообщением об ошибке (получены данные не в формате json). Корректный запрос к сервису имеет следующий формат:
```json
{
    "urls": [
        "https://example.com/",
        "https://example.org/",
        "https://example.net/",
        "https://example.edu/"
    ]
}
```
По ключу ```urls``` находится список адресов, к которым надо сделать запрос. Следуя постановке задачи, данных адресов может быть не более 20 (однако в конфигурации может быть указано любое, подробнее об этом в разделе "[Сборка и конфигурация](#Build)").

Если во время обращения по адресам произошел таймаут во время выполнения отдельного запроса или же всего запроса от пользователя в целом, возвращается ответ с кодом 408 и соответствуюшим сообщением об ошибке.

Активных одновременных подключений к сервису может быть не более 100 (однако в конфигурации может быть указано любое, подробнее об этом в разделе "[Сборка и конфигурация](#Build)"). Если пользователь подключится хотя бы 101-ым, он получит ответ с кодом 429 и значением заголовка ```Retry-After: <половина от таймаута запроса одного пользователя>```. Для примера, если таймаут обработки запроса от пользователя - 10 секунд, при неудачном подключении сервис вернет ответ с кодом 429 и заголовком ```Retry-After: 5```.

Максимальный размер тела входящего запроса: мегабайт (сомневаюсь, что с какими-либо параметрами конфигурации можно передать адекватное количество ссылок на мегабайт).

Таймаут обработки запроса пользователем по умолчанию: 10 секунд (однако в конфигурации может быть указано любое, подробнее об этом в разделе "[Сборка и конфигурация](#Build)").

Логично предположить, что если сервису приходит запрос с методом POST, то запросы по заданным адресам нужно делать так же методом POST, так как другой информации кроме адреса в клиентском запросе не предусмотрено, однако обработку дополнительного поля с HTTP-методом в запросе нетрудно внедрить в сервис. Подробнее об этом в разделе "[Перспективы развития](#Perspective)".

При удачном выполнении всех запросов, сервис возвращает json в следующем формате

```json
{
    "responses": [
        {
            "service_url": "https://example.com/",
            "http_status_code": 200,
            "base64_payload": "some base 64 encoded data",
            "content_type": "text/html; charset=UTF-8"
        },
        {
            "service_url": "https://example.org/non_existent_page",
            "http_status_code": 404,
            "base64_payload": "some base 64 encoded data",
            "content_type": "text/html; charset=UTF-8"
        },
        ...
    ]
}
```
Ключ ```responses``` содержит список всех ответов от адресов, к которым делался запрос. Во вложенном в список объекте по ключу ```service_url``` находится адрес, к которому делался запрос, в ```http_status_code``` - код ответа, в ```base64_payload``` - тело ответа, закодированное в base64, в ```content_type``` - указание значения заголовка ```Content-Type``` при получении ответа по адресу. Тело ответа закодировано в base64, так как в значении ключа могут находиться не только текстовые данные (в постановке задачи не указано, какого именно формата будут приходить данные, а json - текстовый), поэтому любое полученное тело запроса кодируется в base64 и клиенту сообщается значение заголовка ```Content-Type```, чтобы он мог корректно распарсить данные после декодирования из base64.

## <a name="Build"></a> Сборка и конфигурация
По умолчанию сервис запускается на порту 8080, а запросы принимаются на ```http://localhost:8080/api/mux```. Запуск сервиса осуществляется при помощи docker-compose командой ```docker-compose up```. При желании, можно добавить флаг -d, если не хотите занимать термиинал, но тогда не будет видно логов. О лучшем способе логгирования подробнее в разделе "[Перспективы развития](#Perspective)". По завершении, выполните команду ```docker-compose down```, чтобы изящно завершить работу сервиса и удалить все созданные docker-compose артефакты (контейнеры/сети).

В [docker-compose.yml](https://github.com/Ghytro/simple-http-multiplexer/blob/main/docker-compose.yml) указываются необходимые параметры конфигурации, такие как общий таймаут запроса от пользователя, количество одновременных подключений к серверу, таймаут запроса по одному адресу и прочие ограничения из постановки задачи. В самом [Dockerfile](https://github.com/Ghytro/simple-http-multiplexer/blob/main/Dockerfile) есть несколько этапов сборки. Это необходимо, для того, чтобы при запуске сборки образов с разными параметрами, можно было получить различные образы, например при включении в сборку цели test, в промежуточном контейнере будут запущены тесты. Конечная сборка намного легче [официального образа golang с DockerHub](https://hub.docker.com/_/golang) засчет того, что собранный на предыдущем этапе исполняемый файл копируется в образ на основе Alpine Linux.

## <a name="Perspective"></a> Перспективы развития
В этом разделе описаны улучшения, которые в теории можно включить в данный проект, однако, это будет слишком изощренный сервис для тестового задания.

### Кеширование ответов в Redis
Если размер возвращаемого тела ответа невелик, а данные с запрашиваемого ресурса редко обновляются, можно кешировать запросы в Redis для того, чтобы ускорить выдачу ответов на запросы. Прирост производительности будет особенно заметен, если ресурс будет часто запрашиваемым.

### Добавление большего количества полей в запрос от пользователя
К каждому запросу помимо url можно добавить дополнительную информацию в роде версии HTTP, значения заголовков и HTTP-метод, для того чтобы запросы были более точные.

### Использование сторонних библиотек для более элегантного кода
1. Роутер из библиотеки [gorilla/mux](https://github.com/gorilla/mux) обладает намного бо́льшими возможностями, что делает возможным более точную обработку приходящих запросов исходя из большего числа параметров HTTP-запроса;
2. Во внешней стандартной библиотеке Go [golang.org/x/time/rate](https://pkg.go.dev/golang.org/x/time/rate) есть несколько типов рейт-лимитеров, в том числе по количеству одновременных подключений к серверу;
3. Более элегантный хендлинг ошибок из горутин при помощи ErrorGroup из внешней стандартной библиотеки Go [golang.org/x/sync/errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup)

### Усовершенствование логгирования
Для того, чтобы логи не терялись в stdout или при запуске docker-compose с флагом ```-d```, можно прикрепить том к контейнеру и писать логи туда. Также можно настроить логгирование в БД.
