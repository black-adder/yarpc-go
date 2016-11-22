// Code generated by thriftrw v1.0.0
// @generated

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

package service

import (
	"errors"
	"fmt"
	"go.uber.org/thriftrw/wire"
	"strings"
)

type Service_Procedure_Args struct {
	Argument string `json:"argument"`
}

func (v *Service_Procedure_Args) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	w, err = wire.NewValueString(v.Argument), error(nil)
	if err != nil {
		return w, err
	}
	fields[i] = wire.Field{ID: 1, Value: w}
	i++
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *Service_Procedure_Args) FromWire(w wire.Value) error {
	var err error
	argumentIsSet := false
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 1:
			if field.Value.Type() == wire.TBinary {
				v.Argument, err = field.Value.GetString(), error(nil)
				if err != nil {
					return err
				}
				argumentIsSet = true
			}
		}
	}
	if !argumentIsSet {
		return errors.New("field Argument of Service_Procedure_Args is required")
	}
	return nil
}

func (v *Service_Procedure_Args) String() string {
	var fields [1]string
	i := 0
	fields[i] = fmt.Sprintf("Argument: %v", v.Argument)
	i++
	return fmt.Sprintf("Service_Procedure_Args{%v}", strings.Join(fields[:i], ", "))
}

func (v *Service_Procedure_Args) MethodName() string {
	return "procedure"
}

func (v *Service_Procedure_Args) EnvelopeType() wire.EnvelopeType {
	return wire.Call
}

var Service_Procedure_Helper = struct {
	Args           func(argument string) *Service_Procedure_Args
	IsException    func(error) bool
	WrapResponse   func(string, error) (*Service_Procedure_Result, error)
	UnwrapResponse func(*Service_Procedure_Result) (string, error)
}{}

func init() {
	Service_Procedure_Helper.Args = func(argument string) *Service_Procedure_Args {
		return &Service_Procedure_Args{Argument: argument}
	}
	Service_Procedure_Helper.IsException = func(err error) bool {
		switch err.(type) {
		default:
			return false
		}
	}
	Service_Procedure_Helper.WrapResponse = func(success string, err error) (*Service_Procedure_Result, error) {
		if err == nil {
			return &Service_Procedure_Result{Success: &success}, nil
		}
		return nil, err
	}
	Service_Procedure_Helper.UnwrapResponse = func(result *Service_Procedure_Result) (success string, err error) {
		if result.Success != nil {
			success = *result.Success
			return
		}
		err = errors.New("expected a non-void result")
		return
	}
}

type Service_Procedure_Result struct {
	Success *string `json:"success,omitempty"`
}

func (v *Service_Procedure_Result) ToWire() (wire.Value, error) {
	var (
		fields [1]wire.Field
		i      int = 0
		w      wire.Value
		err    error
	)
	if v.Success != nil {
		w, err = wire.NewValueString(*(v.Success)), error(nil)
		if err != nil {
			return w, err
		}
		fields[i] = wire.Field{ID: 0, Value: w}
		i++
	}
	if i != 1 {
		return wire.Value{}, fmt.Errorf("Service_Procedure_Result should have exactly one field: got %v fields", i)
	}
	return wire.NewValueStruct(wire.Struct{Fields: fields[:i]}), nil
}

func (v *Service_Procedure_Result) FromWire(w wire.Value) error {
	var err error
	for _, field := range w.GetStruct().Fields {
		switch field.ID {
		case 0:
			if field.Value.Type() == wire.TBinary {
				var x string
				x, err = field.Value.GetString(), error(nil)
				v.Success = &x
				if err != nil {
					return err
				}
			}
		}
	}
	count := 0
	if v.Success != nil {
		count++
	}
	if count != 1 {
		return fmt.Errorf("Service_Procedure_Result should have exactly one field: got %v fields", count)
	}
	return nil
}

func (v *Service_Procedure_Result) String() string {
	var fields [1]string
	i := 0
	if v.Success != nil {
		fields[i] = fmt.Sprintf("Success: %v", *(v.Success))
		i++
	}
	return fmt.Sprintf("Service_Procedure_Result{%v}", strings.Join(fields[:i], ", "))
}

func (v *Service_Procedure_Result) MethodName() string {
	return "procedure"
}

func (v *Service_Procedure_Result) EnvelopeType() wire.EnvelopeType {
	return wire.Reply
}