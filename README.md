# Protobus
Библиотека для работы с Kafka

## Содержание
- [Формат данных](#формат-данных)
- [Маршаллер](#маршаллер)
- [Producer](#producer)
- [Consumer](#consumer-group)
  - [Handlers](#handlers)
  - [Commit offset](#commit-offset)
  - [Обработка ошибок](#обработка-ошибок)
  - [Ребалансировка](#ребалансировка)
- [Метрики](#метрики)
  

## Формат данных
Сообщения, которые необходимо отправлять должны импортировать [event.proto](event/event.proto),
в нем содержатся свойства:
1. topic (string) - обязательное, связывает событие с топиком Kafka;
2. key (bool) - обязательное для идемпотентного продюсера, по ключевому полю в Kafka определяется partition key (сообщения с одинаковым ключом попадают в одну и ту же партицию).

```protobuf
syntax = "proto3";
package events;

import "event/event.proto";

message Event {
  option (event.topic) = "test_topic";

  int64 id = 1 [ (event.key) = true ];
  string result = 2;
}
```

## Маршаллер
На данный момент не используется schema registry, так как это добавляет нагрузку на devops.  

Вместо schema registry тип сообщения записывается в заголовок Kafka.  
В продюсере и консьюмере маршаллер представлен как метод имеющий сигнатуры:
```go
type (
	MarshalFunc func(proto.Message) ([]byte, error)
    UnmarshalFunc func([]byte, proto.Message) error
)
```
По умолчанию используется стандартный proto.Marshal и proto.Unmarshal, но можно поменять через опции (например если есть потребность работать с json).  
Unmarshal метод можно указать индивидуально на хендлер через опцию:
```go
c.AddHandler(consumer.NewHandlerProto(
    func(ctx context.Context, msg *TestEvent1, raw consumer.Message) error {
        return nil
    }, consumer.WithHandlerJSONUnmarshal()))
```


## Producer
### Конструктор
Конструктор имеет следующую сигнатуру
```go
NewProducer(brokers string, opts ...Option) (*Producer, error)
```
и создает Producer со следующими свойствами по-умолчанию:
- queue.buffering.max.messages = 500000. Размер C буфера для сообщений.
- [acks](https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#acks) = all. Сообщение считается успешно доставленным, когда синхронные реплики подтвердили доставку;
- [enable.idempotence](https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#enable-idempotence) = true. Включена идемпотентная отправка;
- [linger.ms](https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#linger-ms) = 0. Задержка перед отправкой сообщения из буфера;
- [batch.size](https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#batch-size) = 16384. Размер буфера;
- [max.in.flight.requests.per.connection](https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#max-in-flight-requests-per-connection) = 5. Количество одновременно отправляемых сообщений, для идемпотентности возможно максимальное ограничение - 5.
- [retries](https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#retries) = 2147483647 (максимально возможное). Количество повторов при ошибке отправки;
- [retry.backoff.ms](https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#retry-backoff-ms) = 100. Через какое время повторить отправку;
- retry.backoff.max.ms = 10000. Максимальное время повтора отправки.

#### Опции
В конструктор необходимо передать опции, если требуется изменить значения по-умолчанию.  
Пример конструктора со всеми доступными опциями:
```go
producer, err := protobus.NewProducer(
    brokers,
    protobus.WithLogger(zerolog.New(os.Stdout)),
	protobus.WithMarshaller(yourOwnMarshaller),
	protobus.WithSasl("mechanism","username","password"),
    protobus.WithLinger(10),
    protobus.WithBatchSize(8096),
	protobus.WithoutIdempotence(),
	protobus.WithRetries(100),
    protobus.WithRetryBackOff(500),
)
```

#### Отправка сообщения
Любой из методов использует буферы Kafka библиотеки,
фактическая отправка происходит при достижении одного из двух условий:
1. По времени (параметр [linger.ms](https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#linger-ms)), по умолчанию 0 (немедленная отправка);
2. По размеру буфера (параметр [batch.size](https://docs.confluent.io/platform/current/installation/configuration/producer-configs.html#batch-size)), по умолчанию 16384.

Изменение данных параметров через опции может существенно повысить производительность,
но возникает риск потери сообщений.

Producer имеет два метода для отправки сообщений:
- Асинхронный. Добавляет сообщение в буфер Kafka библиотеки и не ждет фактической отправки
  ```go
  func (p *Producer) Send(msg proto.Message) error
  ```
- Синхронный. Добавляет сообщение в буфер и ждет отчета о доставки из Kafka
  ```go
  func (p *Producer) SendSync(msg proto.Message) error
  ```
  
Перед тем как погасить сервис необходимо обязательно закрыть продюсер чтобы он завершил отправку всех сообщений из буфера.

#### Пример использования
```go
producer, err := protobus.NewProducer(
    brokers,
    protobus.WithLogger(zerolog.New(os.Stdout)),
    protobus.WithLinger(10),
    protobus.WithBatchSize(8096),
)

if err != nil {
    return err
}

err = producer.Send(&TestEvent1{
    Id:  uuid.NewString(),
    Num: rand.Int64(),
    Nm: &NestedMsg{
        Name:     "nested_name",
        Duration: rand.Uint64(),
    },
})

if err != nil {
    return err
}

err = producer.SendSync(&TestEvent1{
    Id:  uuid.NewString(),
    Num: rand.Int64(),
    Nm: &NestedMsg{
        Name:     "nested_name",
        Duration: rand.Uint64(),
    },
})
if err != nil {
    return err
}

err = producer.Close()
if err != nil {
    return err
}
```

## Consumer group
### Конструктор
Конструктор имеет следующую сигнатуру
```go
func NewConsumerGroup(brokers string, group string, opts ...Option) (*ConsumerGroup, error)
```
и создает экземпляр ConsumerGroup со следующими свойствами Kafka по-умолчанию:  
- [enable.auto.commit](https://docs.confluent.io/platform/current/installation/configuration/consumer-configs.html#enable-auto-commit) = false. Автокоммит отключен тк коммиты контролируются пакетом;
- [session.timeout.ms](https://docs.confluent.io/platform/current/installation/configuration/consumer-configs.html#session-timeout-ms) = 10000. Таймаут, после которого клиент считается отключенным и происходит ребалансировка;
- [heartbeat.interval.ms](https://docs.confluent.io/platform/current/installation/configuration/consumer-configs.html#heartbeat-interval-ms) = 3000;
- [auto.offset.reset](https://docs.confluent.io/platform/current/installation/configuration/consumer-configs.html#auto-offset-reset) = "latest". При первом подключении получаем новые сообщения (а не весь топик);
- [fetch.min.bytes](https://docs.confluent.io/platform/current/installation/configuration/consumer-configs.html#fetch-min-bytes) = 1. Минимальное количество данных перед отправкой клиенту (меньшее значение снижает задержку обработки, увеличенное улучшает пропускную способность);
- [fetch.wait.max.ms](https://docs.confluent.io/platform/current/installation/configuration/consumer-configs.html#fetch-max-wait-ms) = 500. Максимальное время, которое брокер ждет перед отправкой данных клиенту,  

а также опцииями клиента:
- TopicQueueSize = 50. Размер канала топика;
- TopicCommitCount = 20. Количество сообщений, через которое произойдет коммит офсетов;
- TopicCommitDuration = time.Second. Время, через которое произойдет коммит оффсетов;
- CommitQueueSize = 50. Размер канала для коммитов.

От данных параметров сильно зависит производительность.

Пример, параметры для синхронного потребителя:
- TopicCommitCount = 1;
- TopicQueueSize = 0;
- CommitQueueSize = 0;

### Опции
```go
consumer, err := protobus.NewConsumerGroup(
    brokers,
    protobus.WithLogger(zerolog.New(os.Stdout)),
    protobus.WithMarshaller(yourOwnMarshaller),
    protobus.WithSasl("mechanism","username","password"),
    protobus.WithEarliestOffsetReset(),
    protobus.WithFetchMaxWait(1000),
    protobus.WithFetchMinBytes(2048),
    protobus.WithErrHandler(myErrorHandler),
    protobus.WithTopicQueueSize(100),
    protobus.WithTopicCommitCount(50),
    protobus.WithTopicCommitDuration(0),
    protobus.WithCommitQueueSize(100),
)
```

#### Сахар
Для оркестрации потребителей есть также фасад для ConsumerGroup,
который контролирует клиентов в зависимости от переданных групп в обработчики.

Пример использования обработчиков с одинаковым типов сообщения, но с разными группами:
```go
handlerBatch := func(msgs []*TestEvent1, offset protobus.Offset) error {
    //to do something
    return nil
}

handler := func(msgs *TestEvent1, offset protobus.Offset) error {
    //to do something
    return nil
}

consumer := protobus.NewConsumerGroupFacade(brokers)

err := consumer.AddHandler("group1", protobus.NewBatchProtoHandler[*TestEvent1](handlerBatch, 7))
if err != nil {
    return err
}

err := consumer.AddHandler("group2", protobus.NewProtoHandler[*TestEvent1](handler))
if err != nil {
    return err
}
```

### Handlers
Потребитель обрабатывает сообщения топика последовательно, топики обрабатываются параллельно.

Реализовано два типа обработчиков:
- Обработчик сообщения
  ```go
  handlerFunc := func(msg *TestEvent5, off protobus.Offset) (err error) {
   // todo something
  }
  err := consumer.AddHandler(protobus.NewProtoHandler[*TestEvent5](handlerFunc))
  ```
- Обработчик группы сообщений (batch)
  ```go
  handlerFunc := func(msg []*TestEvent5, offse protobus.Offset) (err error) {
    // todo something
  }
  err := consumer.AddHandler(protobus.NewBatchProtoHandler[*TestEvent5](handlerFunc, 10))
  ```
Обработчик группы сообщений вызывается только при полном заполнении буфера, размер которого указан в конструкторе,
если рассматривать пример выше, то он будет срабатывать всегда, когда набирает 10 сообщений.

На один тип события можно добавить несколько обработчиков.  
Такой вариант является валидным:
```go
err := consumer.AddHandler(protobus.NewProtoHandler[*TestEvent5](
    func(msg *TestEvent5, off protobus.Offset) (err error) {
        // todo something
    }))

err := consumer.AddHandler(protobus.NewProtoHandler[*TestEvent5](
    func(msg *TestEvent5, off protobus.Offset) (err error) {
        // todo something else
    }))
```

### Commit offset
Сообщение считается обработанным после выхода из хендлера,
независимо обработалось оно с ошибкой или нет (см. [обработка ошибок](#обработка-ошибок)).  
Обработчик одиночных сообщений сбрасывает офсет к коммиту сразу после выхода из функции.  
Батч-обработчик не передает офсеты до тех пор, пока не сбросит буфер накопленных сообщений.  
Предположим ситуацию, создан потребитель с настройкой коммита каждые 3 сообщения.
Потребитель имеет три обработчика, в котором два из них батчи с буфером из 7 сообщений и 10 сообщений:
```go
err := consumer.AddHandler(protobus.NewProtoHandler[*TestEvent1](
    func(msg *TestEvent1, off protobus.Offset) (err error) {
        // todo something
    }))

err := consumer.AddHandler(protobus.NewBatchProtoHandler[*TestEvent1](
    func(msg *TestEvent1, off protobus.Offset) (err error) {
        // todo something else
    }, 7))

err := consumer.AddHandler(protobus.NewBatchProtoHandler[*TestEvent1](
    func(msg *TestEvent1, off protobus.Offset) (err error) {
        // todo something else
    }, 10))
```

В данном примере мы имеем следующее:
- обработчик 1 может коммитить каждое сообщение;
- обработчик 2 может коммитить каждое 7 сообщение;
- обработчик 3 может коммитить каждое 10 сообщение;
- общая настройка потребителя коммитить каждое 3 сообщение.

Консьюмер придерживается тактики коммитить минимально возможное количество сообщений, чтобы в случае сбоев не потерять их.
В нашем примере первый коммит будет после 12 сообщения и будет подтверждено получение 7 сообщений.  

Данный пример является простым так как события имеют один и тот же тип, с разнородными сообщениями мы можем получить
другую конфигурацию.  

Общее правило: Если вы остерегаетесь получения дублей сообщений при перезапуске, ребалансе или сбое сервиса
не смешивайте в одной группе батч-обработчики с обработчиками одиночных сообщений. Лучше всего батч-обработчики вешать
обособленно от остальных.

### Обработка ошибок
В конструкторе можно передать обработчик ошибок, который имеет следующую сигнатуру:
```go
func(Message, error)
```

Функция вызывается если есть проблемы с десериализацией или произошла ошибка в обработчике.
В этом месте можно ее корректно обработать, например поместить сообщение в отдельный топик для анализа
или где-то уведомить ответственного (алерт).

Типы ошибок:
```go
errHandler := func(msg protobus.Message, err error) {
    if errors.Is(err, protobus.ErrMessageHandle) {
        // ошибка произошла во время обработки сообщения         		
    }

    if errors.Is(err, protobus.ErrMessageUnmarshal) {
        // маршаллер не смог десериализовать сообщение 
    }
}
```
Структура Message имеет внутри полный контекст пришедшего сообщения. Для батч-обработчиков Message, 
это последнее пришедшее сообщение.

### Ребалансировка
- Добавление партиций к обработке  
Происходит после подписки на топик и старта обработки, но до того как начнут приходит первые сообщения.   
- Удаление партиций из обработки  
Происходит перед закрытием потребителя или если Kafka решает, что можно выполнить более лучший баланс чтения сообщений.
В этом случае коммитятся текущие оффсеты и партиция удаляется из обработки.
Нужно иметь в виду, что в буферах могут быть необработанные сообщения и эти сообщения будут обработаны дважды.

## Метрики
Soon