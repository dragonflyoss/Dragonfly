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

package exception

// AuthError is an interface that represents an error caused by an authentication failure.
type AuthError struct {
}

func (authError *AuthError) Error() string {
	return "NOT AUTH"
}

// IsAuthError is to judge whether an error is AuthError.
func IsAuthError(err error) bool {
	if _, ok := err.(*AuthError); ok {
		return true
	}
	return false
}
