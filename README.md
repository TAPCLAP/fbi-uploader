# fbi-uploader

CLI-утилиты для загрузки бандлов Facebook Instant Games через [двухшаговый Graph API flow](https://developers.facebook.com/docs/games/build/instant-games/get-started/test-publish-share/#api-functionality).

В одном модуле два бинарника:

| Бинарник | Назначение |
|----------|------------|
| `fbi-uploader` | Переупаковка zip с `config.json`, загрузка бандла, опционально push в production |
| `fbi-app-token` | Получение app access token |

## Документация facebook
1. Как заливать zip архив https://developers.facebook.com/docs/games/build/instant-games/get-started/test-publish-share/
1. Как получить app token https://developers.facebook.com/documentation/facebook-login/guides/access-tokens#apptokens

Для заливки бандла нужен `user access token`. Для пуша залитого архива в продакшн, нужен app access token. User access token можно получить вроде только вручную на странице https://developers.facebook.com/tools/accesstoken.

App access token — [client credentials](https://developers.facebook.com/documentation/facebook-login/guides/access-tokens#apptokens)
Его можно получить с помощью утилиты в этом репозитории: `fbi-app-token`:
```bash
export FB_APP_ID=app_id
export FB_APP_SECRET=app_secret
go build -o ./fbi-app-token ./cmd/fbi-app-token
./fbi-app-token
```
Или вручную с помощью запросов:

```bash
export FB_APP_ID=app_id
export FB_APP_SECRET=app_secret
curl -s -G "https://graph.facebook.com/oauth/access_token" \
  --data-urlencode "client_id=${FB_APP_ID}" \
  --data-urlencode "client_secret=${FB_APP_SECRET}" \
  --data-urlencode "grant_type=client_credentials"
```

Ответ — JSON с полем `access_token`. Токен из ответа:

```bash
curl -s -G "https://graph.facebook.com/oauth/access_token" \
  --data-urlencode "client_id=${FB_APP_ID}" \
  --data-urlencode "client_secret=${FB_APP_SECRET}" \
  --data-urlencode "grant_type=client_credentials" \
  | jq -r .access_token
```

Тот же запрос делает утилита [fbi-app-token](#fbi-app-token) в этом репозитории.

## Требования

- **Go 1.25+**

## Сборка

```bash
go build -o fbi-uploader ./cmd/fbi-uploader
go build -o fbi-app-token ./cmd/fbi-app-token
```

## fbi-app-token

Запрашивает app access token по схеме [client credentials](https://developers.facebook.com/documentation/facebook-login/guides/access-tokens#apptokens).

### Переменные окружения

| Переменная | Обязательна | Описание |
|------------|-------------|----------|
| `FB_APP_ID` | да | Meta App ID |
| `FB_APP_SECRET` | да | Meta App Secret |
| `DEBUG` | нет | `true` — отладочные логи в stderr (по умолчанию: `false`) |
| `FB_API_RETRIES` | нет | Максимальное число попыток HTTP-запроса при сетевых ошибках и ответах 5xx (по умолчанию: `10`) |
| `FB_API_RETRY_DELAY_MS` | нет | Начальная пауза между попытками в миллисекундах; удваивается после каждой неудачной попытки (по умолчанию: `1000`) |

### Вывод

При успехе в stdout печатается **только** access token (без перевода строки в конце). При подстановке через command substitution, если shell добавляет `\n`, используйте `tr -d '\n'`:

```bash
./fbi-app-token
```

Ошибки пишутся в stderr; код выхода `1` при сбое.

### Пример

```bash
export FB_APP_ID=123456789
export FB_APP_SECRET=your-app-secret
./fbi-app-token
```

---

## `fbi-uploader`

1. Берёт исходный бандл из `FBINSTANT_ZIP_PATH` (zip-файл) **или** `FBINSTANT_ZIP_PATH_DIR` (папка с файлами). Исходные пути не изменяются.
2. Копирует содержимое во временную директорию (распаковка zip или копирование папки), добавляет `config.json` в корень архива, собирает новый zip в `/tmp/`.
3. Создаёт сессию загрузки и отправляет файл на `rupload.facebook.com` с **user** access token.
4. При `PUSH_TO_PRODUCTION=true` пушит бандл в production (нужен **app** access token).

Сейчас пуш бандла в продакшн (`PUSH_TO_PRODUCTION`) не работает. Не знаю по какой причине. Явно какие-то проблемы на стороне FB. Причем в документации указано, что надо отправлять тело запроса с `{"version_id":"{BUNDLE_INSTANCE_ID}"}` на что API возвращает ошибку, при этом `{"bundle_instance_id":"{BUNDLE_INSTANCE_ID}"}` работает, но не даёт эффекта.

### Переменные окружения — всегда обязательные

| Переменная | Описание |
|------------|----------|
| `FB_APP_ID` | Meta App ID (URL `/uploads` и авторизация при push) |
| `FB_USER_ACCESS_TOKEN` | User access token для `/uploads` и rupload ([Access Token Tool](https://developers.facebook.com/tools/accesstoken), Web Hosting → Get Asset Upload Access Token) |
| `FBINSTANT_ZIP_PATH` **или** `FBINSTANT_ZIP_PATH_DIR` | Путь к исходному `.zip` или к папке с файлами бандла (задаётся ровно одна переменная) |
| `CONFIG_JSON` **или** `CONFIG_JSON_FILE` | JSON inline или путь к содержимому `config.json` (должна быть задана ровно одна переменная) |

### Переменные окружения — только при `PUSH_TO_PRODUCTION=true`

| Переменная | Описание |
|------------|----------|
| `FB_APP_ACCESS_TOKEN` | App access token для `push-to-production` (через `fbi-app-token` или секрет CI) |

Если `PUSH_TO_PRODUCTION` выключен (по умолчанию), `FB_APP_ACCESS_TOKEN` не читается и может быть не задан.

`FB_APP_SECRET` в `fbi-uploader` **не используется**.

app access token обычно возвращется в формате `GG|APP_ID|TOKEN`. Не знаю почему, но `GG|` надо убрать, иначе не работает. То есть для автоматического пуша в продакшн, нужен токен такого вида: `FB_APP_ACCESS_TOKEN=APP_ID|TOKEN`

### config.json

Либо:

```bash
export CONFIG_JSON='{"backendUrl":"https://api.example.com","cdn":"/"}'
```

либо:

```bash
export CONFIG_JSON_FILE=./config.stand.json
```

### Комментарий к бандлу (опционально)

Попадает в заголовок rupload `comment`. Добавляются только непустые переменные:

| Переменная | Фрагмент |
|------------|----------|
| `COMMENT_AREA` | `area: ...` |
| `COMMENT_BACKEND_URL` | `backend_url: ...` |
| `COMMENT_COMMIT` | `commit: ...` |
| `COMMENT_REF` | `ref: ...` |
| `COMMENT_CDN_URL` | `cdn: ...` |
| `COMMENT_EXTRA_INFO` | произвольный текст в конце без префикса `key:` |

Пример результата:

```text
area: stand, backend_url: https://api.example.dev, commit: abc123, ref: main, cdn: https://cdn.example.dev/
```

### Прочие опциональные переменные

| Переменная | По умолчанию | Описание |
|------------|--------------|----------|
| `PUSH_TO_PRODUCTION` | `false` | `true`, `1` или `yes` — вызвать push-to-production после upload |
| `FB_GRAPH_API_VERSION` | `v24.0` | Версия Graph API для сессии загрузки |
| `FB_API_RETRIES` | `10` | Максимальное число попыток HTTP-запроса к Facebook API при сетевых ошибках (timeout, разрыв соединения и т.п.) и ответах 5xx. Для upload бандла каждая попытка создаёт новую upload-сессию |
| `FB_API_RETRY_DELAY_MS` | `1000` | Начальная пауза между попытками в миллисекундах; удваивается после каждой неудачной попытки (1 с → 2 с → 4 с → …) |
| `DEBUG` | `false` | Отладочные логи в stderr |

---

## Примеры

### Только upload (stand / CI)

App token не нужен.

```bash
export FB_APP_ID=123456789
export FB_USER_ACCESS_TOKEN=EAA...
export FBINSTANT_ZIP_PATH=./build/game.zip
# или: export FBINSTANT_ZIP_PATH_DIR=./build/game
export CONFIG_JSON_FILE=./config.stand.json

export COMMENT_AREA=stand
export COMMENT_BACKEND_URL=https://api.example.dev
export COMMENT_COMMIT=$CI_COMMIT_SHA
export COMMENT_REF=$CI_COMMIT_REF_NAME
export COMMENT_CDN_URL=https://cdn.example.dev/

./fbi-uploader
```

### Upload и push в production

```bash
export FB_APP_ID=123456789
export FB_USER_ACCESS_TOKEN=EAA...
export FBINSTANT_ZIP_PATH=./build/game.zip
export CONFIG_JSON_FILE=./config.prod.json
export PUSH_TO_PRODUCTION=true

export FB_APP_ACCESS_TOKEN=$(FB_APP_ID=$FB_APP_ID FB_APP_SECRET=$FB_APP_SECRET ./fbi-app-token | tr -d '\n')

./fbi-uploader
```

### Фрагмент CI pipeline

```bash
# Сборка game.zip выполняется отдельно, затем:
export FB_APP_ID="${FBINSTANT_APP_ID}"
export FB_USER_ACCESS_TOKEN="${FBINSTANT_USER_TOKEN}"
export FBINSTANT_ZIP_PATH="${CI_PROJECT_DIR}/build/fb-instant.zip"
export CONFIG_JSON_FILE="${CI_PROJECT_DIR}/config.fb.json"
export COMMENT_AREA="${AREA}"
export COMMENT_COMMIT="${CI_COMMIT_SHA}"
export COMMENT_REF="${CI_COMMIT_REF_NAME}"

if [ "${PUSH_TO_PRODUCTION}" = "true" ]; then
  export FB_APP_ACCESS_TOKEN=$(FB_APP_SECRET="${FBINSTANT_APP_SECRET}" ./fbi-app-token | tr -d '\n')
fi

./fbi-uploader
```

---

## Docker

Dockerfile: `docker/fbi-uploader/Dockerfile`. Сборка локально:

```bash
docker build -f docker/fbi-uploader/Dockerfile -t fbi-uploader:local .
```

Образы для CI публикуются в GHCR workflow'ами из `.github/workflows/`.

