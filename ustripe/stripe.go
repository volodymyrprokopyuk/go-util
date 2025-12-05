package ustripe

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/stripe/stripe-go/v82"
	"github.com/stripe/stripe-go/v82/webhook"
	"github.com/urfave/cli/v3"
	"github.com/volodymyrprokopyuk/go-util/ucheck"
)

type disableStripeLogger struct{}

func (l *disableStripeLogger) Debugf(format string, args ...any) {}
func (l *disableStripeLogger) Warnf(format string, args ...any) {}
func (l *disableStripeLogger) Infof(format string, args ...any) {}
func (l *disableStripeLogger) Errorf(format string, args ...any) {}

func NewClient(stripeKey string) (*stripe.Client, error) {
  stripe.DefaultLeveledLogger = &disableStripeLogger{}
  stp := stripe.NewClient(stripeKey)
  return stp, nil
}

func Error(err error) error {
  serr, assert := err.(*stripe.Error)
  if !assert {
    return err
  }
  return fmt.Errorf("%s", serr.Msg)
}

func ReadEvent(r *http.Request, whSecret string) (*stripe.Event, error) {
  body, err := io.ReadAll(r.Body)
  if err != nil {
    return nil, err
  }
  ev, err := webhook.ConstructEvent(
    body, r.Header.Get("Stripe-Signature"), whSecret,
  )
  if err != nil {
    return nil, Error(err)
  }
  return &ev, nil
}

func WebhookDelete(
  ctx context.Context, stp *stripe.Client, id string,
) (*stripe.WebhookEndpoint, error) {
  wh, err := stp.V1WebhookEndpoints.Delete(ctx, id, nil)
  if err != nil {
    return nil, Error(err)
  }
  return wh, nil
}

func webhookDeleteAction(
  stripeKey string,
) func(ctx context.Context, cmd *cli.Command) error {
  return func(ctx context.Context, cmd *cli.Command) error {
    // Arguments
    id := cmd.String("id")
    if !ucheck.CheckIDMinPrefix(id, 20, "we_") {
      return errors.New("valid webhook ID must be provided")
    }
    // Stripe
    stp, err := NewClient(stripeKey)
    if err != nil {
      return err
    }
    wh, err := WebhookDelete(ctx, stp, id)
    if err != nil {
      return err
    }
    fmt.Printf("=> webhook %s %s\n", wh.ID, "deleted")
    return nil
  }
}

func WebhookDeleteCmd(stripeKey string) *cli.Command {
  cmd := &cli.Command{
    Name: "delete",
    Usage: "Delete Stripe webhook by ID",
    Action: webhookDeleteAction(stripeKey),
  }
  cmd.Flags = []cli.Flag{
    &cli.StringFlag{
      Name: "id", Usage: "webhook ID", Required: true,
    },
  }
  return cmd
}
