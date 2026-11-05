import { useState, useRef, useEffect, useCallback } from 'react';
import { Input, Button, Typography, Spin, List, Popconfirm, message } from 'antd';
import {
  SendOutlined,
  PlusOutlined,
  DeleteOutlined,
  RobotOutlined,
  UserOutlined,
  MessageOutlined,
} from '@ant-design/icons';
import ReactMarkdown from 'react-markdown';
import remarkGfm from 'remark-gfm';
import { sessionApi } from '../../services/api';
import type { ChatSession, ChatMessageItem } from '../../services/api';

const { Text } = Typography;

export default function ChatPage() {
  const [sessions, setSessions] = useState<ChatSession[]>([]);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [messages, setMessages] = useState<ChatMessageItem[]>([]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const [sessionsLoading, setSessionsLoading] = useState(true);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = useCallback(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, []);

  useEffect(() => { scrollToBottom(); }, [messages, scrollToBottom]);

  const loadSessions = useCallback(async () => {
    try {
      const res = await sessionApi.list();
      setSessions(res.data || []);
    } catch {
      setSessions([]);
    } finally {
      setSessionsLoading(false);
    }
  }, []);

  useEffect(() => { loadSessions(); }, [loadSessions]);

  const selectSession = useCallback(async (id: string) => {
    setActiveId(id);
    try {
      const res = await sessionApi.get(id);
      setMessages(res.data.messages || []);
    } catch {
      setMessages([]);
    }
  }, []);

  // 会话列表加载完成后默认打开第一个会话；删除当前会话后也自动切到下一个。
  useEffect(() => {
    if (sessionsLoading || sessions.length === 0) return;
    if (activeId && sessions.some(s => s.id === activeId)) return;
    const first = sessions[0];
    if (first) selectSession(first.id);
  }, [sessionsLoading, sessions, activeId, selectSession]);

  const createSession = async () => {
    try {
      const res = await sessionApi.create();
      setSessions(prev => [res.data, ...prev]);
      setActiveId(res.data.id);
      setMessages([]);
    } catch {
      message.error('创建会话失败');
    }
  };

  const deleteSession = async (id: string) => {
    try {
      await sessionApi.delete(id);
      setSessions(prev => prev.filter(s => s.id !== id));
      if (activeId === id) {
        setActiveId(null);
        setMessages([]);
      }
      message.success('已删除');
    } catch {
      message.error('删除失败');
    }
  };

  const sendMessage = async () => {
    const text = input.trim();
    if (!text || loading || !activeId) return;

    const tempUser: ChatMessageItem = {
      id: Date.now(),
      session_id: activeId,
      role: 'user',
      content: text,
      created_at: new Date().toISOString(),
    };
    setMessages(prev => [...prev, tempUser]);
    setInput('');
    setLoading(true);

    try {
      const res = await sessionApi.chat(activeId, text);
      const tempAssistant: ChatMessageItem = {
        id: Date.now() + 1,
        session_id: activeId,
        role: 'assistant',
        content: res.data.reply,
        created_at: new Date().toISOString(),
      };
      setMessages(prev => [...prev, tempAssistant]);

      if (messages.length === 0) {
        loadSessions();
      }
    } catch (err) {
      const detail = (err as { response?: { data?: { error?: string } } }).response?.data?.error;
      setMessages(prev => [
        ...prev,
        {
          id: Date.now() + 1,
          session_id: activeId,
          role: 'assistant',
          content: detail || 'AI 助手暂不可用，请确认后端已配置 DEEPSEEK_API_KEY。',
          created_at: new Date().toISOString(),
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ display: 'flex', height: 'calc(100vh - 160px)', background: '#fff', borderRadius: 8, overflow: 'hidden', border: '1px solid #f0f0f0' }}>
      {/* 会话列表侧边栏 */}
      <div style={{ width: 260, borderRight: '1px solid #f0f0f0', display: 'flex', flexDirection: 'column', background: '#fafafa' }}>
        <div style={{ padding: '16px', borderBottom: '1px solid #f0f0f0' }}>
          <Button type="primary" icon={<PlusOutlined />} block onClick={createSession}>
            新建对话
          </Button>
        </div>
        <div style={{ flex: 1, overflow: 'auto' }}>
          {sessionsLoading ? (
            <div style={{ textAlign: 'center', padding: 40 }}><Spin /></div>
          ) : sessions.length === 0 ? (
            <div style={{ textAlign: 'center', color: '#999', padding: 40, fontSize: 13 }}>
              暂无对话，点击上方按钮新建
            </div>
          ) : (
            <List
              dataSource={sessions}
              renderItem={(s) => (
                <List.Item
                  onClick={() => selectSession(s.id)}
                  style={{
                    padding: '10px 16px',
                    cursor: 'pointer',
                    background: activeId === s.id ? '#e6f4ff' : 'transparent',
                    borderLeft: activeId === s.id ? '3px solid #1677ff' : '3px solid transparent',
                    transition: 'all 0.2s',
                  }}
                  actions={[
                    <Popconfirm
                      key="del"
                      title="确认删除此对话？"
                      onConfirm={(e) => { e?.stopPropagation(); deleteSession(s.id); }}
                      onCancel={(e) => e?.stopPropagation()}
                    >
                      <Button
                        type="text"
                        size="small"
                        danger
                        icon={<DeleteOutlined />}
                        onClick={(e) => e.stopPropagation()}
                      />
                    </Popconfirm>,
                  ]}
                >
                  <List.Item.Meta
                    avatar={<MessageOutlined style={{ fontSize: 16, color: '#1677ff', marginTop: 4 }} />}
                    title={<Text ellipsis style={{ maxWidth: 140, fontSize: 13 }}>{s.title}</Text>}
                    description={<Text type="secondary" style={{ fontSize: 11 }}>{new Date(s.updated_at).toLocaleString('zh-CN')}</Text>}
                  />
                </List.Item>
              )}
            />
          )}
        </div>
      </div>

      {/* 对话区域 */}
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column' }}>
        {activeId ? (
          <>
            <div style={{ flex: 1, overflow: 'auto', padding: '16px 24px' }}>
              {messages.length === 0 && !loading && (
                <div style={{ textAlign: 'center', color: '#999', marginTop: 100 }}>
                  <RobotOutlined style={{ fontSize: 48, marginBottom: 16 }} />
                  <div style={{ fontSize: 16 }}>你好！我是学生管理系统的 AI 助手</div>
                  <div style={{ marginTop: 8, color: '#bbb' }}>
                    你可以问我关于学生信息、课程成绩、GPA 分析等问题
                  </div>
                  <div style={{ marginTop: 16, color: '#bbb', fontSize: 13 }}>
                    例如："查一下张三的成绩"、"分析高等数学的成绩分布"、"哪些学生的 GPA 低于 2.0"
                  </div>
                </div>
              )}
              {messages.map((msg) => (
                <div
                  key={msg.id}
                  style={{
                    display: 'flex',
                    justifyContent: msg.role === 'user' ? 'flex-end' : 'flex-start',
                    marginBottom: 16,
                  }}
                >
                  <div
                    style={{
                      maxWidth: '70%',
                      padding: '10px 16px',
                      borderRadius: 12,
                      background: msg.role === 'user' ? '#1677ff' : '#f5f5f5',
                      color: msg.role === 'user' ? '#fff' : '#333',
                    }}
                  >
                    <div style={{ fontSize: 12, marginBottom: 4, opacity: 0.7 }}>
                      {msg.role === 'user' ? <><UserOutlined /> 你</> : <><RobotOutlined /> AI 助手</>}
                    </div>
                    {msg.role === 'user' ? (
                      <Text style={{ color: 'inherit', whiteSpace: 'pre-wrap' }}>{msg.content}</Text>
                    ) : (
                      <div className="markdown-body">
                        <ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown>
                      </div>
                    )}
                  </div>
                </div>
              ))}
              {loading && (
                <div style={{ display: 'flex', justifyContent: 'flex-start', marginBottom: 16 }}>
                  <div style={{ padding: '10px 16px', borderRadius: 12, background: '#f5f5f5' }}>
                    <Spin size="small" /> <Text type="secondary">思考中...</Text>
                  </div>
                </div>
              )}
              <div ref={messagesEndRef} />
            </div>
            <div style={{ padding: '12px 24px', borderTop: '1px solid #f0f0f0' }}>
              <div style={{ display: 'flex', gap: 8 }}>
                <Input
                  value={input}
                  onChange={(e) => setInput(e.target.value)}
                  onPressEnter={sendMessage}
                  placeholder="输入你的问题..."
                  size="large"
                  disabled={loading}
                  style={{ flex: 1 }}
                />
                <Button
                  type="primary"
                  size="large"
                  icon={<SendOutlined />}
                  onClick={sendMessage}
                  loading={loading}
                >
                  发送
                </Button>
              </div>
            </div>
          </>
        ) : (
          <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
            <div style={{ textAlign: 'center', color: '#999' }}>
              <RobotOutlined style={{ fontSize: 64, marginBottom: 16, color: '#d9d9d9' }} />
              <div style={{ fontSize: 16, marginBottom: 8 }}>选择一个对话或新建对话开始</div>
              <Button type="primary" icon={<PlusOutlined />} onClick={createSession}>
                新建对话
              </Button>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
