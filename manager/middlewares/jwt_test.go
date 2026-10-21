/*
 *     Copyright 2020 The Dragonfly Authors
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

package middlewares

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"

	"d7y.io/dragonfly/v2/manager/config"
	"d7y.io/dragonfly/v2/manager/models"
	"d7y.io/dragonfly/v2/manager/service/mocks"
)

func TestJwtLoginCookieHTTPOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	svc := mocks.NewMockService(ctrl)
	svc.EXPECT().SignIn(gomock.Any(), gomock.Any()).Return(&models.User{BaseModel: models.BaseModel{ID: 1}}, nil)

	authMiddleware, err := Jwt(config.JWTConfig{
		Realm:      "Dragonfly",
		Key:        "testsecretkeytestsecretkey",
		Timeout:    time.Hour,
		MaxRefresh: time.Hour,
	}, svc)
	assert.NoError(t, err)

	r := gin.New()
	r.POST("/users/signin", authMiddleware.LoginHandler)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/users/signin", strings.NewReader(`{"name":"foobar","password":"password123"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var jwtCookie *http.Cookie
	for _, cookie := range w.Result().Cookies() {
		if cookie.Name == "jwt" {
			jwtCookie = cookie
			break
		}
	}

	assert.NotNil(t, jwtCookie, "login must set the jwt cookie")
	assert.True(t, jwtCookie.HttpOnly, "jwt session cookie must be HttpOnly so page scripts cannot read the token")
}
