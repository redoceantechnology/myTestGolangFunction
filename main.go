package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/valyala/fastjson"

	"log"
	//"time"

	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"
	"go.opentelemetry.io/otel"

	//"go.opentelemetry.io/otel/attribute"
	//"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"
	"go.opentelemetry.io/otel/trace"
)

func HandleRequest(request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	ApiResponse := events.APIGatewayProxyResponse{}
	// Switch for identifying the HTTP request
	switch request.HTTPMethod {
	case "GET":
		// Obtain the QueryStringParameter
		number, _ := strconv.ParseUint(request.QueryStringParameters["number"], 10, 32)

		if number != 0 {
			fibo, _ := Fibonacci(uint(number))

			ApiResponse = events.APIGatewayProxyResponse{Body: "Hey " + strconv.FormatUint(fibo, 10) + " welcome! ", StatusCode: 200}
		} else {
			ApiResponse = events.APIGatewayProxyResponse{Body: "Error: Query Parameter name missing", StatusCode: 500}
		}

	case "POST":
		//validates json and returns error if not working
		err := fastjson.Validate(request.Body)

		if err != nil {
			body := "Error: Invalid JSON payload ||| " + fmt.Sprint(err) + " Body Obtained" + "||||" + request.Body
			ApiResponse = events.APIGatewayProxyResponse{Body: body, StatusCode: 500}
		} else {
			ApiResponse = events.APIGatewayProxyResponse{Body: request.Body, StatusCode: 200}
		}

	}
	// Response

	return ApiResponse, nil
}

// Fibonacci returns the n-th fibonacci number.
func Fibonacci(n uint) (uint64, error) {

	if n <= 1 {
		return uint64(n), nil
	}

	if n > 93 {
		return 0, fmt.Errorf("unsupported fibonacci number %d: too large", n)
	}

	var n2, n1 uint64 = 0, 1
	for i := uint(2); i < n; i++ {
		n2, n1 = n1, n1+n2
	}

	return n2 + n1, nil
}

func main() {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String("go-quickstart"), //TODO Replace with the name of your application
			semconv.ServiceVersionKey.String("1.0.1"),      //TODO Replace with the version of your application
		),
	)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	exporter, err := otlptracehttp.New(
		ctx,
		otlptracehttp.WithEndpoint(os.Getenv("URL")),                                                         //TODO Replace <URL> to your SaaS/Managed-URL as mentioned in the next step
		otlptracehttp.WithURLPath(os.Getenv("URL_PATH")),                                                     //TODO Replace <URL_PATH> to your SaaS/Managed-URL-PATH as mentioned in the next step
		otlptracehttp.WithHeaders(map[string]string{"Authorization": "Api-Token " + os.Getenv("API_TOKEN")}), //TODO Replace <TOKEN> with your API Token as mentioned in the next step
	)
	if err != nil {
		log.Fatalf("Failed to create OTLP exporter: %v", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Printf("Error shutting down tracer provider: %v", err)
		}
	}()
	var span trace.Span
	ctx, span = otel.Tracer("go-quickstart").Start(ctx, "lambda.Start")
	lambda.Start(otellambda.InstrumentHandler(HandleRequest, xrayconfig.WithRecommendedOptions(tp)...))
	defer span.End()
}
