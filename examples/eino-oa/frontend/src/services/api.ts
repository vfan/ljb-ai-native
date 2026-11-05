import axios from 'axios';

const api = axios.create({
  baseURL: '/api',
  timeout: 10000,
});

// 401 时通知全局 AuthProvider 回到登录页；登录接口本身除外。
api.interceptors.response.use(
  (response) => response,
  (error) => {
    const url = String(error.config?.url || '');
    if (error.response?.status === 401 && !url.includes('/auth/login')) {
      window.dispatchEvent(new Event('eo:unauthorized'));
    }
    return Promise.reject(error);
  },
);

export interface PaginatedResult<T> {
  items: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface Student {
  id?: number;
  student_no: string;
  name: string;
  gender: string;
  age: number;
  phone: string;
  email: string;
  department: string;
  major: string;
  class_name: string;
  enrollment_year: number;
  status: string;
  created_at?: string;
  updated_at?: string;
}

export interface Course {
  id?: number;
  course_code: string;
  name: string;
  credits: number;
  department: string;
  teacher: string;
  semester: string;
  description: string;
  created_at?: string;
  updated_at?: string;
}

export interface Grade {
  id?: number;
  student_id: number;
  course_id: number;
  score: number;
  grade_point?: number;
  semester: string;
  status: string;
  student?: Student;
  course?: Course;
  created_at?: string;
  updated_at?: string;
}

export interface GradeStats {
  course_id: number;
  course_name: string;
  avg_score: number;
  max_score: number;
  min_score: number;
  pass_rate: number;
  count: number;
}

export interface StudentGPA {
  student_id: number;
  student_no: string;
  student_name: string;
  gpa: number;
  total_courses: number;
}

export const studentApi = {
  list: (params?: Record<string, unknown>) =>
    api.get<PaginatedResult<Student>>('/students', { params }),
  get: (id: number) => api.get<Student>(`/students/${id}`),
  create: (data: Student) => api.post<Student>('/students', data),
  update: (id: number, data: Student) => api.put<Student>(`/students/${id}`, data),
  delete: (id: number) => api.delete(`/students/${id}`),
};

export const courseApi = {
  list: (params?: Record<string, unknown>) =>
    api.get<PaginatedResult<Course>>('/courses', { params }),
  get: (id: number) => api.get<Course>(`/courses/${id}`),
  create: (data: Course) => api.post<Course>('/courses', data),
  update: (id: number, data: Course) => api.put<Course>(`/courses/${id}`, data),
  delete: (id: number) => api.delete(`/courses/${id}`),
};

export const gradeApi = {
  list: (params?: Record<string, unknown>) =>
    api.get<PaginatedResult<Grade>>('/grades', { params }),
  get: (id: number) => api.get<Grade>(`/grades/${id}`),
  create: (data: Grade) => api.post<Grade>('/grades', data),
  update: (id: number, data: Grade) => api.put<Grade>(`/grades/${id}`, data),
  delete: (id: number) => api.delete(`/grades/${id}`),
  statsByCourse: (courseId: number) => api.get<GradeStats>(`/grades/stats/course/${courseId}`),
  calculateGPA: (studentId: number) => api.get<StudentGPA>(`/grades/gpa/student/${studentId}`),
  academicWarning: (threshold?: number) =>
    api.get<{ threshold: number; students: StudentGPA[] }>('/grades/warning', {
      params: { threshold },
    }),
};

export interface ChatSession {
  id: string;
  user_id?: number;
  title: string;
  created_at: string;
  updated_at: string;
}

export interface ChatMessageItem {
  id: number;
  session_id: string;
  role: 'user' | 'assistant';
  content: string;
  created_at: string;
}

export interface ChatSessionDetail extends ChatSession {
  messages: ChatMessageItem[];
}

export const sessionApi = {
  list: () => api.get<ChatSession[]>('/sessions'),
  get: (id: string) => api.get<ChatSessionDetail>(`/sessions/${id}`),
  create: (title?: string) => api.post<ChatSession>('/sessions', { title }),
  delete: (id: string) => api.delete(`/sessions/${id}`),
  chat: (id: string, message: string) =>
    api.post<{ reply: string }>(`/sessions/${id}/chat`, { message }, { timeout: 120000 }),
};

// ---------- 鉴权 ----------

export interface CurrentUser {
  id: number;
  username: string;
  display_name: string;
  role: string;
  auth_method: 'session' | 'pat' | 'oauth';
  scopes: string[];
}

export interface AccessToken {
  id: number;
  name: string;
  kind: string;
  prefix: string;
  scopes: string;
  expires_at: string | null;
  last_used_at: string | null;
  revoked_at: string | null;
  created_at: string;
}

export const authApi = {
  login: (username: string, password: string) =>
    api.post<{ user: CurrentUser }>('/auth/login', { username, password }),
  logout: () => api.post('/auth/logout'),
  me: () => api.get<CurrentUser>('/auth/me'),
};

export const tokenApi = {
  list: () => api.get<AccessToken[]>('/auth/tokens'),
  create: (data: { name: string; scopes: string; expires_in_days: number }) =>
    api.post<{ token: string; item: AccessToken }>('/auth/tokens', data),
  revoke: (id: number) => api.delete(`/auth/tokens/${id}`),
  revokeOAuth: () => api.delete<{ message: string; revoked: number }>('/auth/oauth-tokens'),
};

export default api;
