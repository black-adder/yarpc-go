// Copyright (c) 2016 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package transport_test

import (
	"context"
	"testing"
	"time"

	"go.uber.org/yarpc/api/transport"
	"go.uber.org/yarpc/internal/request"

	"github.com/stretchr/testify/assert"
)

func TestValidator(t *testing.T) {
	tests := []struct {
		req           *transport.Request
		transportType transport.Type
		ttl           time.Duration

		wantErr string
	}{
		{
			// No error
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Encoding:  "raw",
				Procedure: "hello",
			},
			ttl: time.Second,
		},
		{
			// encoding is not required
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "hello",
			},
			wantErr: "missing encoding",
		},
		{
			req: &transport.Request{
				Service:   "service",
				Procedure: "hello",
				Encoding:  "raw",
			},
			wantErr: "missing caller name",
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Procedure: "hello",
				Encoding:  "raw",
			},
			wantErr: "missing service name",
		},
		{
			req: &transport.Request{
				Caller:   "caller",
				Service:  "service",
				Encoding: "raw",
			},
			wantErr: "missing procedure",
		},
		{
			req: &transport.Request{
				Caller:    "caller",
				Service:   "service",
				Procedure: "hello",
				Encoding:  "raw",
			},
			transportType: transport.Unary,
			wantErr:       "missing TTL",
		},
		{
			req:     &transport.Request{},
			wantErr: "missing service name, procedure, caller name, and encoding",
		},
	}

	for _, tt := range tests {
		ctx := context.Background()
		err := transport.ValidateRequest(tt.req)

		if err == nil && tt.transportType == transport.Unary {
			var cancel func()

			if tt.ttl != 0 {
				ctx, cancel = context.WithTimeout(ctx, tt.ttl)
				defer cancel()
			}

			err = request.ValidateUnaryContext(ctx)
		}

		if tt.wantErr != "" {
			if assert.Error(t, err) {
				assert.Equal(t, tt.wantErr, err.Error())
			}
		} else {
			assert.NoError(t, err)
		}
	}
}
