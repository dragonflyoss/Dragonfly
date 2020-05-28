/*
 * Copyright The Dragonfly Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package api

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"testing"

	"github.com/stretchr/testify/suite"
)

func TestSuite(t *testing.T) {
	suite.Run(t, new(APISuite))
}

type APISuite struct {
	suite.Suite
	validHandler   *HandlerSpec
	invalidHandler *HandlerSpec
}

func (s *APISuite) SetupSuite() {
	s.validHandler = &HandlerSpec{
		Method: "GET",
		HandlerFunc: func(context.Context, http.ResponseWriter, *http.Request) error {
			return nil
		},
	}
}

func (s *APISuite) SetupTest() {
	for _, v := range apiCategories {
		v.handlerSpecs = nil
	}
}

func (s *APISuite) TestCategory_Register() {
	var cases = []struct {
		c *category
		h *HandlerSpec
	}{
		{V1, s.invalidHandler},
		{V1, s.validHandler},
		{Extension, s.validHandler},
		{Legacy, s.validHandler},
	}

	for _, v := range cases {
		before := v.c.handlerSpecs
		v.c.Register(v.h)
		after := v.c.handlerSpecs
		if s.invalidHandler == v.h {
			s.Equal(before, after)
		} else if s.validHandler == v.h {
			s.Equal(len(before)+1, len(after))
			s.Equal(after[len(after)-1], v.h)
		}
	}
}

func (s *APISuite) TestCategory_others() {
	for k, v := range apiCategories {
		s.Equal(k, v.name)
		s.Equal(v.Name(), v.name)
		s.Equal(v.Prefix(), v.prefix)
		s.Equal(v.Handlers(), v.handlerSpecs)
	}
}

func (s *APISuite) TestNewCategory() {
	// don't create a category with the same name
	for k, v := range apiCategories {
		c := newCategory(k, "")
		s.Equal(c, v)
	}

	// don't create a category with empty name
	c := newCategory("", "x")
	s.Nil(c)

	// create a new category
	name := fmt.Sprintf("%v", rand.Float64())
	c = newCategory(name, "/")
	defer delete(apiCategories, name)
	s.Equal(c.name, name)
	s.Equal(c.prefix, "/")
}
