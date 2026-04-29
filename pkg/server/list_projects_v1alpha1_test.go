package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	svcv1alpha1 "github.com/akuity/kargo/api/service/v1alpha1"
	kargoapi "github.com/akuity/kargo/api/v1alpha1"
	"github.com/akuity/kargo/pkg/server/config"
	"github.com/akuity/kargo/pkg/server/kubernetes"
	"github.com/akuity/kargo/pkg/server/user"
)

func TestListProjects(t *testing.T) {
	testCases := map[string]struct {
		objects    []client.Object
		cfg        config.ServerConfig
		userInfo   *user.Info
		req        *svcv1alpha1.ListProjectsRequest
		assertions func(*testing.T, *connect.Response[svcv1alpha1.ListProjectsResponse], error)
	}{
		"no projects": {
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Empty(t, r.Msg.GetProjects())
			},
		},
		"orders by name": {
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "z-project"}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "a-project"}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "m-project"}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "0-project"}},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetProjects(), 4)

				// Check that the projects are ordered by name.
				require.Equal(t, "0-project", r.Msg.GetProjects()[0].GetName())
				require.Equal(t, "a-project", r.Msg.GetProjects()[1].GetName())
				require.Equal(t, "m-project", r.Msg.GetProjects()[2].GetName())
				require.Equal(t, "z-project", r.Msg.GetProjects()[3].GetName())
			},
		},
		"mine filters to mapped projects": {
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "project-a"}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "project-b"}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "project-c"}},
			},
			userInfo: &user.Info{
				ServiceAccountsByNamespace: map[string]map[types.NamespacedName]struct{}{
					"project-a": {
						{Namespace: "project-a", Name: "viewer"}: {},
					},
					"project-c": {
						{Namespace: "project-c", Name: "admin"}: {},
					},
				},
			},
			req: &svcv1alpha1.ListProjectsRequest{
				Mine: ptr.To(true),
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.NotNil(t, r)
				require.Len(t, r.Msg.GetProjects(), 2)
				require.Equal(t, "project-a", r.Msg.GetProjects()[0].GetName())
				require.Equal(t, "project-c", r.Msg.GetProjects()[1].GetName())
			},
		},
		"labels filter matches projects": {
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "project-a",
					Labels: map[string]string{"team.io/name": "platform"},
				}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "project-b",
					Labels: map[string]string{"team.io/name": "backend"},
				}},
			},
			cfg: config.ServerConfig{
				ProjectLabelPrefixes: []string{"team.io/"},
			},
			req: &svcv1alpha1.ListProjectsRequest{
				Labels: []string{"team.io/name=platform"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.Len(t, r.Msg.GetProjects(), 1)
				require.Equal(t, "project-a", r.Msg.GetProjects()[0].GetName())
				require.Equal(t, int32(1), r.Msg.GetTotal())
			},
		},
		"labels filter no match returns empty": {
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "project-a",
					Labels: map[string]string{"team.io/name": "platform"},
				}},
			},
			cfg: config.ServerConfig{
				ProjectLabelPrefixes: []string{"team.io/"},
			},
			req: &svcv1alpha1.ListProjectsRequest{
				Labels: []string{"team.io/name=nonexistent"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.Empty(t, r.Msg.GetProjects())
				require.Equal(t, int32(0), r.Msg.GetTotal())
			},
		},
		"labels filter uses AND logic": {
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name: "project-a",
					Labels: map[string]string{
						"team.io/name": "platform",
						"team.io/env":  "prod",
					},
				}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "project-b",
					Labels: map[string]string{"team.io/name": "platform"},
				}},
			},
			cfg: config.ServerConfig{
				ProjectLabelPrefixes: []string{"team.io/"},
			},
			req: &svcv1alpha1.ListProjectsRequest{
				Labels: []string{
					"team.io/name=platform",
					"team.io/env=prod",
				},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.Len(t, r.Msg.GetProjects(), 1)
				require.Equal(t, "project-a", r.Msg.GetProjects()[0].GetName())
			},
		},
		"available_labels computed before label filter": {
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "project-a",
					Labels: map[string]string{"team.io/name": "platform"},
				}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "project-b",
					Labels: map[string]string{"team.io/name": "backend"},
				}},
			},
			cfg: config.ServerConfig{
				ProjectLabelPrefixes: []string{"team.io/"},
			},
			req: &svcv1alpha1.ListProjectsRequest{
				Labels: []string{"team.io/name=platform"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.Len(t, r.Msg.GetProjects(), 1)
				require.Equal(t, []string{
					"team.io/name=backend",
					"team.io/name=platform",
				}, r.Msg.GetAvailableLabels())
			},
		},
		"available_labels only includes prefix-matching labels": {
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name: "project-a",
					Labels: map[string]string{
						"team.io/name":        "platform",
						"unrelated/something": "ignored",
					},
				}},
			},
			cfg: config.ServerConfig{
				ProjectLabelPrefixes: []string{"team.io/"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.Equal(t, []string{"team.io/name=platform"}, r.Msg.GetAvailableLabels())
			},
		},
		"available_labels empty when no prefixes configured": {
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "project-a",
					Labels: map[string]string{"team.io/name": "platform"},
				}},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.Empty(t, r.Msg.GetAvailableLabels())
			},
		},
		"labels filter composes with name filter": {
			objects: []client.Object{
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "alpha-platform",
					Labels: map[string]string{"team.io/name": "platform"},
				}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "alpha-backend",
					Labels: map[string]string{"team.io/name": "backend"},
				}},
				&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{
					Name:   "beta-platform",
					Labels: map[string]string{"team.io/name": "platform"},
				}},
			},
			cfg: config.ServerConfig{
				ProjectLabelPrefixes: []string{"team.io/"},
			},
			req: &svcv1alpha1.ListProjectsRequest{
				Filter: ptr.To("alpha"),
				Labels: []string{"team.io/name=platform"},
			},
			assertions: func(t *testing.T, r *connect.Response[svcv1alpha1.ListProjectsResponse], err error) {
				require.NoError(t, err)
				require.Len(t, r.Msg.GetProjects(), 1)
				require.Equal(t, "alpha-platform", r.Msg.GetProjects()[0].GetName())
				require.Equal(t, int32(1), r.Msg.GetTotal())
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			if testCase.userInfo != nil {
				ctx = user.ContextWithInfo(ctx, *testCase.userInfo)
			}

			client, err := kubernetes.NewClient(
				ctx,
				&rest.Config{},
				kubernetes.ClientOptions{
					SkipAuthorization: true,
					NewInternalClient: func(
						_ context.Context,
						_ *rest.Config,
						scheme *runtime.Scheme,
					) (client.WithWatch, error) {
						c := fake.NewClientBuilder().WithScheme(scheme)
						if len(testCase.objects) > 0 {
							c.WithObjects(testCase.objects...)
						}
						return c.Build(), nil
					},
				},
			)
			require.NoError(t, err)

			req := testCase.req
			if req == nil {
				req = &svcv1alpha1.ListProjectsRequest{}
			}
			svr := &server{client: client, cfg: testCase.cfg}
			res, err := svr.ListProjects(ctx, &connect.Request[svcv1alpha1.ListProjectsRequest]{
				Msg: req,
			})
			testCase.assertions(t, res, err)
		})
	}
}

func Test_server_listProjects(t *testing.T) {
	testRESTEndpoint(
		t, &config.ServerConfig{},
		http.MethodGet, "/v1beta1/projects",
		[]restTestCase{
			{
				name: "no Projects exist",
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					list := &kargoapi.ProjectList{}
					err := json.Unmarshal(w.Body.Bytes(), list)
					require.NoError(t, err)
					require.Empty(t, list.Items)
				},
			},
			{
				name: "lists Projects",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "z-project"}},
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "a-project"}},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)

					// Examine the Projects in the response
					projects := &kargoapi.ProjectList{}
					err := json.Unmarshal(w.Body.Bytes(), projects)
					require.NoError(t, err)
					require.Len(t, projects.Items, 2)
					require.Equal(t, "a-project", projects.Items[0].Name)
					require.Equal(t, "z-project", projects.Items[1].Name)
				},
			},
			{
				name: "mine=true without user info returns empty",
				url:  "/v1beta1/projects?mine=true",
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "project-a"}},
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "project-b"}},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					projects := &kargoapi.ProjectList{}
					err := json.Unmarshal(w.Body.Bytes(), projects)
					require.NoError(t, err)
					require.Empty(t, projects.Items)
				},
			},
			{
				name: "mine=true filters to mapped projects",
				url:  "/v1beta1/projects?mine=true",
				ctxSetup: func(ctx context.Context) context.Context {
					return user.ContextWithInfo(
						ctx,
						user.Info{
							ServiceAccountsByNamespace: map[string]map[types.NamespacedName]struct{}{
								"project-a": {{Namespace: "project-a", Name: "viewer"}: {}},
							},
						},
					)
				},
				clientBuilder: fake.NewClientBuilder().WithObjects(
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "project-a"}},
					&kargoapi.Project{ObjectMeta: metav1.ObjectMeta{Name: "project-b"}},
				),
				assertions: func(t *testing.T, w *httptest.ResponseRecorder, _ client.Client) {
					require.Equal(t, http.StatusOK, w.Code)
					projects := &kargoapi.ProjectList{}
					err := json.Unmarshal(w.Body.Bytes(), projects)
					require.NoError(t, err)
					require.Len(t, projects.Items, 1)
					require.Equal(t, "project-a", projects.Items[0].Name)
				},
			},
		},
	)
}

func Test_collectAvailableLabels(t *testing.T) {
	testCases := []struct {
		name     string
		projects []kargoapi.Project
		prefixes []string
		expected []string
	}{
		{
			name:     "nil prefixes returns nil",
			projects: []kargoapi.Project{{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"k": "v"}}}},
			expected: nil,
		},
		{
			name:     "empty prefixes returns nil",
			projects: []kargoapi.Project{{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"k": "v"}}}},
			prefixes: []string{},
			expected: nil,
		},
		{
			name: "collects matching labels sorted and deduped",
			projects: []kargoapi.Project{
				{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					"team.io/name": "platform",
					"team.io/env":  "prod",
					"other/key":    "ignored",
				}}},
				{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					"team.io/name": "backend",
				}}},
				{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					"team.io/name": "platform",
				}}},
			},
			prefixes: []string{"team.io/"},
			expected: []string{
				"team.io/env=prod",
				"team.io/name=backend",
				"team.io/name=platform",
			},
		},
		{
			name: "multiple prefixes",
			projects: []kargoapi.Project{
				{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{
					"team.io/name": "platform",
					"org.io/dept":  "eng",
				}}},
			},
			prefixes: []string{"team.io/", "org.io/"},
			expected: []string{
				"org.io/dept=eng",
				"team.io/name=platform",
			},
		},
		{
			name:     "no projects returns empty",
			projects: nil,
			prefixes: []string{"team.io/"},
			expected: []string{},
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := collectAvailableLabels(tc.projects, tc.prefixes)
			if tc.expected == nil {
				require.Nil(t, result)
			} else {
				require.Equal(t, tc.expected, result)
			}
		})
	}
}

func Test_filterProjectsByLabels(t *testing.T) {
	projects := []kargoapi.Project{
		{ObjectMeta: metav1.ObjectMeta{
			Name: "project-a",
			Labels: map[string]string{
				"team.io/name": "platform",
				"team.io/env":  "prod",
			},
		}},
		{ObjectMeta: metav1.ObjectMeta{
			Name:   "project-b",
			Labels: map[string]string{"team.io/name": "backend"},
		}},
		{ObjectMeta: metav1.ObjectMeta{
			Name: "project-c",
		}},
	}
	testCases := []struct {
		name       string
		wantLabels []string
		expected   []string
	}{
		{
			name:       "single label match",
			wantLabels: []string{"team.io/name=platform"},
			expected:   []string{"project-a"},
		},
		{
			name:       "multiple labels AND logic",
			wantLabels: []string{"team.io/name=platform", "team.io/env=prod"},
			expected:   []string{"project-a"},
		},
		{
			name:       "no match",
			wantLabels: []string{"team.io/name=nonexistent"},
			expected:   nil,
		},
		{
			name:       "partial match fails AND",
			wantLabels: []string{"team.io/name=backend", "team.io/env=prod"},
			expected:   nil,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := filterProjectsByLabels(projects, tc.wantLabels)
			names := make([]string, len(result))
			for i, p := range result {
				names[i] = p.Name
			}
			if tc.expected == nil {
				require.Empty(t, result)
			} else {
				require.Equal(t, tc.expected, names)
			}
		})
	}
}

func Test_projectHasAllLabels(t *testing.T) {
	testCases := []struct {
		name       string
		labels     map[string]string
		wantLabels []string
		expected   bool
	}{
		{
			name:       "all labels present",
			labels:     map[string]string{"a": "1", "b": "2"},
			wantLabels: []string{"a=1", "b=2"},
			expected:   true,
		},
		{
			name:       "missing key",
			labels:     map[string]string{"a": "1"},
			wantLabels: []string{"a=1", "b=2"},
			expected:   false,
		},
		{
			name:       "wrong value",
			labels:     map[string]string{"a": "1", "b": "3"},
			wantLabels: []string{"a=1", "b=2"},
			expected:   false,
		},
		{
			name:       "nil labels",
			labels:     nil,
			wantLabels: []string{"a=1"},
			expected:   false,
		},
		{
			name:       "empty want labels",
			labels:     map[string]string{"a": "1"},
			wantLabels: nil,
			expected:   true,
		},
		{
			name:       "label with empty value",
			labels:     map[string]string{"a": ""},
			wantLabels: []string{"a="},
			expected:   true,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, tc.expected, projectHasAllLabels(tc.labels, tc.wantLabels))
		})
	}
}

func Test_filterProjectsByAccess(t *testing.T) {
	testProjects := []kargoapi.Project{
		{ObjectMeta: metav1.ObjectMeta{Name: "project-a"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "project-b"}},
		{ObjectMeta: metav1.ObjectMeta{Name: "project-c"}},
	}
	testCases := []struct {
		name     string
		userInfo *user.Info
		assert   func(*testing.T, []kargoapi.Project)
	}{
		{
			name: "no user info in context",
			assert: func(t *testing.T, result []kargoapi.Project) {
				require.Empty(t, result)
			},
		},
		{
			name:     "admin user with no SA mappings",
			userInfo: &user.Info{IsAdmin: true},
			assert: func(t *testing.T, result []kargoapi.Project) {
				require.Empty(t, result)
			},
		},
		{
			name:     "user with nil SA map",
			userInfo: &user.Info{},
			assert: func(t *testing.T, result []kargoapi.Project) {
				require.Empty(t, result)
			},
		},
		{
			name: "OIDC user with matching project namespaces",
			userInfo: &user.Info{
				ServiceAccountsByNamespace: map[string]map[types.NamespacedName]struct{}{
					"project-a": {
						{Namespace: "project-a", Name: "viewer"}: {},
					},
					"project-c": {
						{Namespace: "project-c", Name: "admin"}: {},
					},
				},
			},
			assert: func(t *testing.T, result []kargoapi.Project) {
				require.Len(t, result, 2)
				require.Equal(t, "project-a", result[0].Name)
				require.Equal(t, "project-c", result[1].Name)
			},
		},
		{
			name: "OIDC user with no matching namespaces",
			userInfo: &user.Info{
				ServiceAccountsByNamespace: map[string]map[types.NamespacedName]struct{}{
					"kargo": {
						{Namespace: "kargo", Name: "viewer"}: {},
					},
				},
			},
			assert: func(t *testing.T, result []kargoapi.Project) {
				require.Empty(t, result)
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			if testCase.userInfo != nil {
				ctx = user.ContextWithInfo(ctx, *testCase.userInfo)
			}
			result := filterProjectsByAccess(ctx, testProjects)
			testCase.assert(t, result)
		})
	}
}
