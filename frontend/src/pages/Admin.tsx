import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { 
  ArrowLeft, 
  RefreshCw, 
  Server, 
  Settings, 
  BookOpen, 
  Clock, 
  Sparkles, 
  CheckCircle2, 
  AlertTriangle,
  LayoutDashboard,
  ChevronLeft,
  ChevronRight,
  Search,
  TrendingUp,
  AlertCircle,
  Plus,
  X,
  Bot,
  Trash2,
  MessageSquare
} from 'lucide-react';
import { useTenant } from '../context/TenantContext';
import { searchApi, adminApi, AISuggestion, ProposedAction, Conversation } from '../services/api';
import Footer from '../components/Footer';

interface SyncLog {
  timestamp: string;
  tenantId: string;
  tenantName: string;
  syncedCount: number;
  status: 'SUCCESS' | 'FAILED';
  errorMessage?: string;
}

interface ChatMessageItem {
  id?: string;
  role: 'user' | 'assistant' | 'system';
  content: string;
  proposed_actions?: ProposedAction[];
  action_states?: Record<string, string>;
}

const Admin: React.FC = () => {
  const navigate = useNavigate();
  const { activeTenant, setActiveTenantById, tenants } = useTenant();

  // Navigation & Layout states
  const [activeTab, setActiveTab] = useState<'dashboard' | 'ai-suggestions' | 'dictionaries' | 'sync' | 'ai-assistant'>('dashboard');
  const [sidebarCollapsed, setSidebarCollapsed] = useState(false);

  // Search Analytics Dashboard states
  const [analyticsSummary, setAnalyticsSummary] = useState<any>(null);
  const [zeroResultQueries, setZeroResultQueries] = useState<any[]>([]);
  const [categoryAnalytics, setCategoryAnalytics] = useState<any[]>([]);
  const [analyticsRange, setAnalyticsRange] = useState<string>('30days');
  const [loadingAnalytics, setLoadingAnalytics] = useState(false);
  const [triggeringAnalytics, setTriggeringAnalytics] = useState(false);
  const [triggerStartDate, setTriggerStartDate] = useState('');
  const [triggerEndDate, setTriggerEndDate] = useState('');

  // Sync state
  const [syncing, setSyncing] = useState(false);
  const [syncMessage, setSyncMessage] = useState<{ type: 'success' | 'error'; text: string } | null>(null);
  const [syncLogs, setSyncLogs] = useState<SyncLog[]>([]);

  // AI Suggestions lists and total counts from Backend (independent for Typo and Synonym tables)
  const [typoSuggestions, setTypoSuggestions] = useState<AISuggestion[]>([]);
  const [synonymSuggestions, setSynonymSuggestions] = useState<AISuggestion[]>([]);
  const [typoTotal, setTypoTotal] = useState(0);
  const [synonymTotal, setSynonymTotal] = useState(0);
  const [loadingTypos, setLoadingTypos] = useState(false);
  const [loadingSynonyms, setLoadingSynonyms] = useState(false);

  // Dictionaries lists
  const [spellcheckRules, setSpellcheckRules] = useState<Array<{ id: string; typo_word: string; correct_word: string; status: string }>>([]);
  const [synonymRules, setSynonymRules] = useState<Array<{ id: string; keyword: string; synonym: string; status: string }>>([]);
  const [loadingDictionaries, setLoadingDictionaries] = useState(false);

  // Independent search & pagination states mapping directly to Backend params
  const [typoStatus, setTypoStatus] = useState<'pending' | 'approved' | 'rejected'>('pending');
  const [synonymStatus, setSynonymStatus] = useState<'pending' | 'approved' | 'rejected'>('pending');
  const [typoSearch, setTypoSearch] = useState('');
  const [synonymSearch, setSynonymSearch] = useState('');
  const [typoPage, setTypoPage] = useState(1);
  const [synonymPage, setSynonymPage] = useState(1);
  const itemsPerPage = 5;

  // Create Manual Dictionary Rules states
  const [showAddRuleModal, setShowAddRuleModal] = useState(false);
  const [newRuleType, setNewRuleType] = useState<'synonym' | 'spellcheck'>('synonym');
  const [newKeyword, setNewKeyword] = useState('');
  const [newSynonym, setNewSynonym] = useState('');
  const [isBidirectional, setIsBidirectional] = useState(false);
  const [newTypoWord, setNewTypoWord] = useState('');
  const [newCorrectWord, setNewCorrectWord] = useState('');
  const [actionLoading, setActionLoading] = useState(false);

  // AI Assistant states
  const [conversations, setConversations] = useState<Conversation[]>([]);
  const [activeConversation, setActiveConversation] = useState<Conversation | null>(null);
  const [loadingConversations, setLoadingConversations] = useState(false);
  const [chatMessages, setChatMessages] = useState<ChatMessageItem[]>([]);
  const [chatInput, setChatInput] = useState('');
  const [sendingChat, setSendingChat] = useState(false);

  const fetchConversations = async () => {
    setLoadingConversations(true);
    try {
      const list = await adminApi.getConversations(activeTenant.id);
      setConversations(list);
      if (list.length > 0) {
        setActiveConversation(list[0]);
      } else {
        setActiveConversation(null);
        setChatMessages([
          {
            role: 'assistant',
            content: 'Xin chào! Tôi là Trợ lý AI của Swift Search Engine. Tôi có thể giúp bạn tra cứu nhanh sản phẩm trong kho hoặc đề xuất điều chỉnh từ điển tìm kiếm (synonym, sửa lỗi chính tả). Bạn cần trợ giúp gì hôm nay?'
          }
        ]);
      }
    } catch (err: any) {
      console.error("Failed to load conversations:", err);
    } finally {
      setLoadingConversations(false);
    }
  };

  useEffect(() => {
    fetchConversations();
  }, [activeTenant.id]);

  useEffect(() => {
    if (!activeConversation) return;

    const loadMessages = async () => {
      try {
        const msgs = await adminApi.getConversationMessages(activeTenant.id, activeConversation.id);
        if (msgs.length > 0) {
          setChatMessages(msgs.map(m => ({
            id: m.id,
            role: m.role,
            content: m.content,
            proposed_actions: m.proposed_actions,
            action_states: m.action_states
          })));
        } else {
          setChatMessages([
            {
              role: 'assistant',
              content: 'Cuộc hội thoại trống. Bạn có thể bắt đầu nhắn tin ngay.'
            }
          ]);
        }
      } catch (err: any) {
        console.error("Failed to load messages:", err);
        setChatMessages([
          {
            role: 'assistant',
            content: `Không thể tải lịch sử tin nhắn: ${err.message}`
          }
        ]);
      }
    };

    loadMessages();
  }, [activeConversation, activeTenant.id]);

  const handleNewConversation = async () => {
    try {
      const newConv = await adminApi.createConversation(activeTenant.id);
      setConversations(prev => [newConv, ...prev]);
      setActiveConversation(newConv);
      setChatMessages([
        {
          role: 'assistant',
          content: 'Cuộc hội thoại mới đã được tạo. Tôi có thể giúp gì cho bạn?'
        }
      ]);
    } catch (err: any) {
      alert(`Không thể tạo cuộc hội thoại mới: ${err.message}`);
    }
  };

  const handleDeleteConversation = async (e: React.MouseEvent, convId: string) => {
    e.stopPropagation();
    if (!window.confirm("Bạn có chắc chắn muốn xóa cuộc hội thoại này?")) return;

    try {
      await adminApi.deleteConversation(activeTenant.id, convId);
      setConversations(prev => prev.filter(c => c.id !== convId));
      if (activeConversation?.id === convId) {
        setActiveConversation(null);
        setChatMessages([
          {
            role: 'assistant',
            content: 'Xin chào! Tôi là Trợ lý AI của Swift Search Engine. Tôi có thể giúp bạn tra cứu nhanh sản phẩm trong kho hoặc đề xuất điều chỉnh từ điển tìm kiếm (synonym, sửa lỗi chính tả). Bạn cần trợ giúp gì hôm nay?'
          }
        ]);
      }
    } catch (err: any) {
      alert(`Không thể xóa cuộc hội thoại: ${err.message}`);
    }
  };

  const handleSendChat = async (e?: React.FormEvent, customMsg?: string) => {
    if (e) e.preventDefault();
    const msgToSend = (customMsg || chatInput).trim();
    if (!msgToSend || sendingChat) return;

    if (!customMsg) {
      setChatInput('');
    }

    setSendingChat(true);
    let currentConv = activeConversation;

    try {
      if (!currentConv) {
        currentConv = await adminApi.createConversation(activeTenant.id, msgToSend.substring(0, 40));
        setConversations(prev => [currentConv!, ...prev]);
        setActiveConversation(currentConv);
      }

      const newUserMsg: ChatMessageItem = {
        role: 'user',
        content: msgToSend
      };

      const prevMsgs = chatMessages.filter(
        m => !m.content.startsWith("Xin chào! Tôi là Trợ lý") && !m.content.startsWith("Cuộc hội thoại")
      );
      const updatedMessages = [...prevMsgs, newUserMsg];
      setChatMessages(updatedMessages);

      const res = await adminApi.chatWithAssistant(activeTenant.id, currentConv.id, msgToSend);

      const newAssistantMsg: ChatMessageItem = {
        id: res.message_id,
        role: 'assistant',
        content: res.reply,
        proposed_actions: res.proposed_actions || [],
        action_states: {}
      };

      setChatMessages([...updatedMessages, newAssistantMsg]);

      if (prevMsgs.length === 0) {
        const updatedList = await adminApi.getConversations(activeTenant.id);
        setConversations(updatedList);
        const match = updatedList.find(c => c.id === currentConv?.id);
        if (match) {
          setActiveConversation(match);
        }
      }
    } catch (err: any) {
      const errorMsg: ChatMessageItem = {
        role: 'assistant',
        content: `Rất tiếc, đã xảy ra lỗi khi kết nối với máy chủ AI: ${err.message || 'Lỗi không xác định'}`
      };
      setChatMessages(prev => [...prev, errorMsg]);
    } finally {
      setSendingChat(false);
    }
  };

  const handleApproveAction = async (msgIndex: number, actionIndex: number, action: ProposedAction) => {
    const msg = chatMessages[msgIndex];
    if (!msg.id) return;
    if (msg.action_states && msg.action_states[String(actionIndex)] === 'accepted') return;

    try {
      if (action.action_type === 'create_synonym') {
        const { keyword, synonym, is_bidirectional } = action.params;
        await adminApi.addSynonym(activeTenant.id, keyword, synonym, is_bidirectional);
      } else if (action.action_type === 'create_spellcheck') {
        const { typo_word, correct_word } = action.params;
        await adminApi.addSpellcheck(activeTenant.id, typo_word, correct_word);
      } else if (action.action_type === 'delete_synonym') {
        const { ids } = action.params;
        for (const id of ids) {
          await adminApi.deleteSynonym(activeTenant.id, id);
        }
      } else if (action.action_type === 'delete_spellcheck') {
        const { ids } = action.params;
        for (const id of ids) {
          await adminApi.deleteSpellcheck(activeTenant.id, id);
        }
      }

      await adminApi.updateActionState(activeTenant.id, msg.id, actionIndex, 'accepted');

      const copy = [...chatMessages];
      if (!copy[msgIndex].action_states) copy[msgIndex].action_states = {};
      copy[msgIndex].action_states![String(actionIndex)] = 'accepted';
      
      const systemMessage: ChatMessageItem = {
        role: 'system',
        content: `Đã áp dụng thành công: ${action.description}`
      };
      
      const newMessages = [...copy, systemMessage];
      setChatMessages(newMessages);

      fetchDictionaries();
    } catch (err: any) {
      alert(`Không thể thực thi hành động: ${err.message || 'Lỗi hệ thống'}`);
    }
  };

  const handleRejectAction = async (msgIndex: number, actionIndex: number, action: ProposedAction) => {
    const msg = chatMessages[msgIndex];
    if (!msg.id) return;

    try {
      await adminApi.updateActionState(activeTenant.id, msg.id, actionIndex, 'rejected');

      const copy = [...chatMessages];
      if (!copy[msgIndex].action_states) copy[msgIndex].action_states = {};
      copy[msgIndex].action_states![String(actionIndex)] = 'rejected';

      const systemMessage: ChatMessageItem = {
        role: 'system',
        content: `Đã từ chối hành động: ${action.description}`
      };

      const newMessages = [...copy, systemMessage];
      setChatMessages(newMessages);
    } catch (err: any) {
      alert(`Không thể từ chối hành động: ${err.message || 'Lỗi hệ thống'}`);
    }
  };

  const handleAddRule = async (e: React.FormEvent) => {
    e.preventDefault();
    setActionLoading(true);
    try {
      if (newRuleType === 'synonym') {
        if (!newKeyword.trim() || !newSynonym.trim()) {
          alert('Vui lòng nhập đầy đủ Từ khóa chính và Từ đồng nghĩa');
          setActionLoading(false);
          return;
        }
        await adminApi.addSynonym(activeTenant.id, newKeyword.trim(), newSynonym.trim(), isBidirectional);
      } else {
        if (!newTypoWord.trim() || !newCorrectWord.trim()) {
          alert('Vui lòng nhập đầy đủ Từ gõ sai và Từ sửa đúng');
          setActionLoading(false);
          return;
        }
        await adminApi.addSpellcheck(activeTenant.id, newTypoWord.trim(), newCorrectWord.trim());
      }
      
      // Reset form states
      setNewKeyword('');
      setNewSynonym('');
      setIsBidirectional(false);
      setNewTypoWord('');
      setNewCorrectWord('');
      setShowAddRuleModal(false);

      // Refresh dictionaries list and search suggestions
      fetchDictionaries();
      fetchTypoSuggestions();
      fetchSynonymSuggestions();
    } catch (err: any) {
      alert(`Thêm quy tắc thất bại: ${err.message || 'Lỗi hệ thống'}`);
    } finally {
      setActionLoading(false);
    }
  };

  const handleDeleteSynonym = async (id: string) => {
    if (!window.confirm('Bạn có chắc chắn muốn xóa quy tắc từ đồng nghĩa này?')) return;
    try {
      await adminApi.deleteSynonym(activeTenant.id, id);
      fetchDictionaries();
      fetchSynonymSuggestions();
    } catch (err: any) {
      alert(`Xóa thất bại: ${err.message || 'Lỗi hệ thống'}`);
    }
  };

  const handleDeleteSpellcheck = async (id: string) => {
    if (!window.confirm('Bạn có chắc chắn muốn xóa quy tắc sửa chính tả này?')) return;
    try {
      await adminApi.deleteSpellcheck(activeTenant.id, id);
      fetchDictionaries();
      fetchTypoSuggestions();
    } catch (err: any) {
      alert(`Xóa thất bại: ${err.message || 'Lỗi hệ thống'}`);
    }
  };

  // Load sync logs from localStorage on mount
  useEffect(() => {
    const savedLogs = localStorage.getItem('search_sync_logs');
    if (savedLogs) {
      try {
        setSyncLogs(JSON.parse(savedLogs));
      } catch (e) {
        console.error('Failed to parse sync logs', e);
      }
    }
  }, []);

  const fetchTypoSuggestions = async () => {
    setLoadingTypos(true);
    try {
      const res = await adminApi.getAISuggestions(
        activeTenant.id,
        typoStatus,
        'typo',
        typoSearch,
        typoPage,
        itemsPerPage
      );
      setTypoSuggestions(res.suggestions);
      setTypoTotal(res.total);
    } catch (e) {
      console.error('Failed to fetch Typo suggestions', e);
    } finally {
      setLoadingTypos(false);
    }
  };

  const fetchSynonymSuggestions = async () => {
    setLoadingSynonyms(true);
    try {
      const res = await adminApi.getAISuggestions(
        activeTenant.id,
        synonymStatus,
        'synonym',
        synonymSearch,
        synonymPage,
        itemsPerPage
      );
      setSynonymSuggestions(res.suggestions);
      setSynonymTotal(res.total);
    } catch (e) {
      console.error('Failed to fetch Synonym suggestions', e);
    } finally {
      setLoadingSynonyms(false);
    }
  };

  const fetchDictionaries = async () => {
    setLoadingDictionaries(true);
    try {
      const spRules = await adminApi.getSpellcheckRules(activeTenant.id);
      setSpellcheckRules(spRules);
      
      const synRules = await adminApi.getSearchSynonyms(activeTenant.id);
      setSynonymRules(synRules);
    } catch (e) {
      console.error('Failed to fetch dictionary rules', e);
    } finally {
      setLoadingDictionaries(false);
    }
  };

  const fetchAnalyticsSummary = async () => {
    setLoadingAnalytics(true);
    try {
      const res = await adminApi.getAnalyticsSummary(activeTenant.id, analyticsRange);
      setAnalyticsSummary(res.summary);
      setZeroResultQueries(res.zero_results || []);
      setCategoryAnalytics(res.category_analytics || []);
    } catch (e) {
      console.error('Failed to fetch analytics summary', e);
    } finally {
      setLoadingAnalytics(false);
    }
  };

  const handleTriggerAggregation = async () => {
    setTriggeringAnalytics(true);
    try {
      if ((triggerStartDate && !triggerEndDate) || (!triggerStartDate && triggerEndDate)) {
        alert('Vui lòng nhập cả Ngày bắt đầu và Ngày kết thúc');
        setTriggeringAnalytics(false);
        return;
      }
      const res = await adminApi.triggerAnalyticsAggregation(activeTenant.id, triggerStartDate, triggerEndDate);
      alert(res.message || 'Đã chạy tổng hợp dữ liệu phân tích thành công!');
      fetchAnalyticsSummary();
    } catch (e) {
      console.error('Failed to trigger aggregation', e);
      alert('Chạy tổng hợp dữ liệu thất bại: ' + e);
    } finally {
      setTriggeringAnalytics(false);
    }
  };

  // Fetch data on active tenant, status, search keyword, or page index changes (Backend-driven)
  useEffect(() => {
    fetchTypoSuggestions();
  }, [activeTenant.id, typoStatus, typoSearch, typoPage]);

  useEffect(() => {
    fetchSynonymSuggestions();
  }, [activeTenant.id, synonymStatus, synonymSearch, synonymPage]);

  useEffect(() => {
    fetchDictionaries();
  }, [activeTenant.id]);

  useEffect(() => {
    if (activeTab === 'dashboard') {
      fetchAnalyticsSummary();
    }
  }, [activeTenant.id, analyticsRange, activeTab]);

  const handleApproveSuggestion = async (id: string, type: 'typo' | 'synonym') => {
    try {
      await adminApi.approveAISuggestion(id, activeTenant.id);
      if (type === 'typo') {
        fetchTypoSuggestions();
      } else {
        fetchSynonymSuggestions();
      }
      fetchDictionaries();
    } catch (e: any) {
      alert(`Phê duyệt thất bại: ${e.message || 'Lỗi hệ thống'}`);
    }
  };

  const handleRejectSuggestion = async (id: string, type: 'typo' | 'synonym') => {
    try {
      await adminApi.rejectAISuggestion(id, activeTenant.id);
      if (type === 'typo') {
        fetchTypoSuggestions();
      } else {
        fetchSynonymSuggestions();
      }
    } catch (e: any) {
      alert(`Từ chối thất bại: ${e.message || 'Lỗi hệ thống'}`);
    }
  };

  const [generatingSuggestions, setGeneratingSuggestions] = useState(false);

  const handleGenerateAISuggestions = async () => {
    if (generatingSuggestions) return;
    setGeneratingSuggestions(true);
    try {
      await adminApi.generateAISuggestions(activeTenant.id);
      // Reset tables page pointers and trigger reload
      setTypoPage(1);
      setSynonymPage(1);
      await Promise.all([fetchTypoSuggestions(), fetchSynonymSuggestions()]);
    } catch (e: any) {
      alert(`Phân tích AI thất bại: ${e.message || 'Lỗi hệ thống'}`);
    } finally {
      setGeneratingSuggestions(false);
    }
  };

  const handleSyncDatabase = async () => {
    setSyncing(true);
    setSyncMessage(null);
    const timestamp = new Date().toLocaleString('vi-VN');

    try {
      const res = await searchApi.syncDatabase(activeTenant.id);
      
      const newLog: SyncLog = {
        timestamp,
        tenantId: activeTenant.id,
        tenantName: activeTenant.name,
        syncedCount: res.synced_count,
        status: 'SUCCESS',
      };

      const updatedLogs = [newLog, ...syncLogs].slice(0, 10);
      setSyncLogs(updatedLogs);
      localStorage.setItem('search_sync_logs', JSON.stringify(updatedLogs));

      setSyncMessage({
        type: 'success',
        text: `Đồng bộ hoàn tất! Đã cập nhật ${res.synced_count} sản phẩm lên OpenSearch.`,
      });
    } catch (err: any) {
      console.error(err);
      const newLog: SyncLog = {
        timestamp,
        tenantId: activeTenant.id,
        tenantName: activeTenant.name,
        syncedCount: 0,
        status: 'FAILED',
        errorMessage: err.message || 'Lỗi kết nối API',
      };

      const updatedLogs = [newLog, ...syncLogs].slice(0, 10);
      setSyncLogs(updatedLogs);
      localStorage.setItem('search_sync_logs', JSON.stringify(updatedLogs));

      setSyncMessage({
        type: 'error',
        text: `Đồng bộ thất bại: ${err.message || 'Không thể kết nối đến Backend Go.'}`,
      });
    } finally {
      setSyncing(false);
    }
  };

  const clearSyncLogs = () => {
    setSyncLogs([]);
    localStorage.removeItem('search_sync_logs');
  };

  // Pagination stats calculation
  const totalTypoPages = Math.ceil(typoTotal / itemsPerPage) || 1;
  const typoStartIndex = (typoPage - 1) * itemsPerPage;

  const totalSynonymPages = Math.ceil(synonymTotal / itemsPerPage) || 1;
  const synonymStartIndex = (synonymPage - 1) * itemsPerPage;

  return (
    <div className="admin-layout">
      {/* COLLAPSIBLE SIDEBAR */}
      <aside className={`admin-sidebar ${sidebarCollapsed ? 'collapsed' : ''}`}>
        <div 
          className="admin-sidebar-header" 
          onClick={() => navigate('/')} 
          style={{ cursor: 'pointer' }}
        >
          <Settings className="text-gradient" size={26} strokeWidth={2.5} style={{ flexShrink: 0 }} />
          <span style={{ fontWeight: 800, fontSize: '1.25rem', letterSpacing: '-0.02em' }} className="admin-menu-text">
            Swift<span style={{ color: 'var(--primary)' }}>Search</span>
          </span>
        </div>

        <button 
          onClick={() => setSidebarCollapsed(!sidebarCollapsed)}
          className="admin-sidebar-toggle"
          title={sidebarCollapsed ? "Mở rộng menu" : "Thu gọn menu"}
        >
          {sidebarCollapsed ? <ChevronRight size={14} /> : <ChevronLeft size={14} />}
        </button>

        <nav className="admin-sidebar-menu">
          <button 
            onClick={() => setActiveTab('dashboard')} 
            className={`admin-menu-item ${activeTab === 'dashboard' ? 'active' : ''}`}
          >
            <LayoutDashboard size={20} />
            <span className="admin-menu-text">Dashboard</span>
          </button>
          
          <button 
            onClick={() => setActiveTab('ai-suggestions')} 
            className={`admin-menu-item ${activeTab === 'ai-suggestions' ? 'active' : ''}`}
          >
            <Sparkles size={20} />
            <span className="admin-menu-text">Đề xuất từ AI</span>
          </button>

          <button 
            onClick={() => setActiveTab('dictionaries')} 
            className={`admin-menu-item ${activeTab === 'dictionaries' ? 'active' : ''}`}
          >
            <BookOpen size={20} />
            <span className="admin-menu-text">Cấu hình Từ khóa</span>
          </button>

          <button 
            onClick={() => setActiveTab('sync')} 
            className={`admin-menu-item ${activeTab === 'sync' ? 'active' : ''}`}
          >
            <RefreshCw size={20} />
            <span className="admin-menu-text">Đồng bộ dữ liệu</span>
          </button>

          <button 
            onClick={() => setActiveTab('ai-assistant')} 
            className={`admin-menu-item ${activeTab === 'ai-assistant' ? 'active' : ''}`}
          >
            <Bot size={20} />
            <span className="admin-menu-text">Trợ lý AI</span>
          </button>
        </nav>

        <div style={{ marginTop: 'auto', padding: '16px' }} className="admin-menu-text">
          <button 
            onClick={() => navigate('/')} 
            className="btn btn-outline" 
            style={{ width: '100%', display: 'flex', gap: '8px', justifyContent: 'center' }}
          >
            <ArrowLeft size={14} />
            Cửa hàng
          </button>
        </div>
      </aside>

      {/* MAIN CONTAINER */}
      <div className="admin-main">
        {/* TOPBAR */}
        <header className="admin-topbar">
          <div>
            <h2 style={{ fontSize: '1.25rem', fontWeight: 800, color: 'var(--text-primary)' }}>
              {activeTab === 'dashboard' && 'Dashboard Analytics'}
              {activeTab === 'ai-suggestions' && 'AI Suggestion Engine'}
              {activeTab === 'dictionaries' && 'Active Search Dictionaries'}
              {activeTab === 'sync' && 'Database Sync Controls'}
              {activeTab === 'ai-assistant' && 'AI Assistant Chat'}
            </h2>
          </div>

          {/* Tenant Switcher in Topbar */}
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <Server size={16} style={{ color: 'var(--text-muted)' }} />
            <span style={{ fontSize: '0.85rem', fontWeight: 600, color: 'var(--text-secondary)' }}>
              Cửa hàng hoạt động:
            </span>
            <select 
              value={activeTenant.id}
              onChange={(e) => setActiveTenantById(e.target.value)}
              className="btn btn-outline"
              style={{ padding: '6px 12px', height: '36px', fontSize: '0.85rem', fontWeight: 600, border: '1.5px solid var(--border-color)', borderRadius: 'var(--radius-sm)' }}
            >
              {tenants.map(t => (
                <option key={t.id} value={t.id}>{t.name}</option>
              ))}
            </select>
          </div>
        </header>

        {/* BODY */}
        <main className="admin-body">
          {/* TAB 1: DASHBOARD (DYNAMIC US-009) */}
          {activeTab === 'dashboard' && (
            <div>
              {/* Range Filter & Trigger aggregation buttons */}
              <div style={{ display: 'flex', justifyContent: 'flex-end', marginBottom: '16px', gap: '8px', alignItems: 'center', flexWrap: 'wrap' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginRight: 'auto' }}>
                  <span style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', fontWeight: 600 }}>Tổng hợp:</span>
                  <input 
                    type="date" 
                    value={triggerStartDate}
                    onChange={(e) => setTriggerStartDate(e.target.value)}
                    className="btn btn-outline"
                    style={{ padding: '4px 8px', fontSize: '0.8rem', height: '32px', border: '1px solid var(--border-color)', borderRadius: 'var(--radius-sm)', backgroundColor: 'var(--bg-primary)', color: 'var(--text-primary)' }}
                  />
                  <span style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>đến</span>
                  <input 
                    type="date" 
                    value={triggerEndDate}
                    onChange={(e) => setTriggerEndDate(e.target.value)}
                    className="btn btn-outline"
                    style={{ padding: '4px 8px', fontSize: '0.8rem', height: '32px', border: '1px solid var(--border-color)', borderRadius: 'var(--radius-sm)', backgroundColor: 'var(--bg-primary)', color: 'var(--text-primary)' }}
                  />
                  <button 
                    onClick={handleTriggerAggregation} 
                    disabled={triggeringAnalytics}
                    className="btn btn-outline"
                    style={{ 
                      padding: '6px 16px', 
                      fontSize: '0.85rem', 
                      fontWeight: 600, 
                      height: '36px',
                      borderColor: 'var(--primary)',
                      color: 'var(--primary)',
                      display: 'flex',
                      alignItems: 'center',
                      gap: '6px'
                    }}
                  >
                    <RefreshCw size={14} className={triggeringAnalytics ? 'spin' : ''} />
                    {triggeringAnalytics ? 'Đang tổng hợp...' : 'Chạy Tổng hợp'}
                  </button>
                </div>
                <button 
                  onClick={() => setAnalyticsRange('today')} 
                  className={`btn ${analyticsRange === 'today' ? 'btn-primary' : 'btn-outline'}`}
                  style={{ padding: '6px 16px', fontSize: '0.85rem', fontWeight: 600, height: '36px' }}
                >
                  Hôm nay
                </button>
                <button 
                  onClick={() => setAnalyticsRange('7days')} 
                  className={`btn ${analyticsRange === '7days' ? 'btn-primary' : 'btn-outline'}`}
                  style={{ padding: '6px 16px', fontSize: '0.85rem', fontWeight: 600, height: '36px' }}
                >
                  7 ngày qua
                </button>
                <button 
                  onClick={() => setAnalyticsRange('30days')} 
                  className={`btn ${analyticsRange === '30days' ? 'btn-primary' : 'btn-outline'}`}
                  style={{ padding: '6px 16px', fontSize: '0.85rem', fontWeight: 600, height: '36px' }}
                >
                  30 ngày qua
                </button>
              </div>

              {loadingAnalytics ? (
                <div style={{ padding: '80px 0', textAlign: 'center', color: 'var(--text-secondary)' }}>
                  <div className="spinner" style={{ margin: '0 auto 16px auto', width: '40px', height: '40px', border: '3px solid rgba(37, 99, 235, 0.1)', borderTop: '3px solid var(--primary)', borderRadius: '50%', animation: 'spin 1s linear infinite' }} />
                  <span>Đang tải dữ liệu phân tích...</span>
                </div>
              ) : (
                <>
                  {/* Premium metrics grid */}
                  <div className="metrics-grid">
                    <div className="metric-card">
                      <div className="metric-card-bg-gradient" />
                      <div className="metric-card-header">
                        <span className="metric-card-title">Tổng số tìm kiếm</span>
                        <div className="metric-card-icon">
                          <TrendingUp size={18} />
                        </div>
                      </div>
                      <div className="metric-card-value">
                        {analyticsSummary ? analyticsSummary.total_searches.toLocaleString() : 0}
                      </div>
                      <div className="metric-card-sub">
                        Lượt tìm kiếm trong {analyticsRange === 'today' ? 'hôm nay' : analyticsRange === '7days' ? '7 ngày qua' : '30 ngày qua'}
                      </div>
                    </div>

                    <div className="metric-card">
                      <div className="metric-card-bg-gradient" />
                      <div className="metric-card-header">
                        <span className="metric-card-title">Tìm kiếm lỗi (0 kết quả)</span>
                        <div className="metric-card-icon" style={{ backgroundColor: 'rgba(239, 68, 68, 0.1)', color: 'var(--danger)' }}>
                          <AlertCircle size={18} />
                        </div>
                      </div>
                      <div className="metric-card-value">
                        {analyticsSummary ? analyticsSummary.zero_result_searches.toLocaleString() : 0}
                      </div>
                      <div className="metric-card-sub">Cần AI phân tích tối ưu hóa</div>
                    </div>

                    <div className="metric-card">
                      <div className="metric-card-bg-gradient" />
                      <div className="metric-card-header">
                        <span className="metric-card-title">Tỷ lệ CTR Tìm kiếm</span>
                        <div className="metric-card-icon" style={{ backgroundColor: 'rgba(16, 185, 129, 0.1)', color: 'var(--success)' }}>
                          <CheckCircle2 size={18} />
                        </div>
                      </div>
                      <div className="metric-card-value">
                        {analyticsSummary ? `${analyticsSummary.ctr}%` : '0%'}
                      </div>
                      <div className="metric-card-sub">
                        Vị trí click TB: {analyticsSummary ? analyticsSummary.avg_click_position : 0}
                      </div>
                    </div>

                    <div className="metric-card">
                      <div className="metric-card-bg-gradient" />
                      <div className="metric-card-header">
                        <span className="metric-card-title">Cấu hình Từ khóa</span>
                        <div className="metric-card-icon" style={{ backgroundColor: 'rgba(37, 99, 235, 0.1)', color: 'var(--primary)' }}>
                          <BookOpen size={18} />
                        </div>
                      </div>
                      <div className="metric-card-value">{spellcheckRules.length + synonymRules.length}</div>
                      <div className="metric-card-sub">{spellcheckRules.length} sửa lỗi, {synonymRules.length} đồng nghĩa</div>
                    </div>
                  </div>

                  {/* Layout for analytics tables */}
                  <div className="admin-grid" style={{ gridTemplateColumns: '1fr 1fr' }}>
                    <section className="admin-card">
                      <div className="admin-card-title">
                        <AlertCircle size={18} style={{ color: 'var(--danger)' }} />
                        <span>Top Từ khóa 0 kết quả (Zero Results)</span>
                      </div>
                      <div className="table-wrapper">
                        <table className="admin-table">
                          <thead>
                            <tr>
                              <th>Từ khóa</th>
                              <th>Số lần tìm</th>
                              <th>Trạng thái đề xuất</th>
                            </tr>
                          </thead>
                          <tbody>
                            {zeroResultQueries.length === 0 ? (
                              <tr>
                                <td colSpan={3} style={{ textAlign: 'center', padding: '24px', color: 'var(--text-secondary)' }}>
                                  Chưa có dữ liệu tìm kiếm lỗi
                                </td>
                              </tr>
                            ) : (
                              zeroResultQueries.map((zq, idx) => (
                                <tr key={idx}>
                                  <td style={{ fontWeight: 600 }}>{zq.query}</td>
                                  <td>{zq.search_count}</td>
                                  <td>
                                    <span 
                                      className="badge-status" 
                                      style={{ 
                                        backgroundColor: zq.ai_suggestion_status === 'Đã gợi ý sửa đổi' ? 'var(--success-light)' : zq.ai_suggestion_status === 'Chờ duyệt' ? '#fef3c7' : '#e5e7eb', 
                                        color: zq.ai_suggestion_status === 'Đã gợi ý sửa đổi' ? 'var(--success)' : zq.ai_suggestion_status === 'Chờ duyệt' ? '#92400e' : 'var(--text-secondary)' 
                                      }}
                                    >
                                      {zq.ai_suggestion_status}
                                    </span>
                                  </td>
                                </tr>
                              ))
                            )}
                          </tbody>
                        </table>
                      </div>
                    </section>

                    <section className="admin-card">
                      <div className="admin-card-title">
                        <TrendingUp size={18} style={{ color: 'var(--primary)' }} />
                        <span>Lượt tìm kiếm phổ biến theo danh mục</span>
                      </div>
                      <div className="table-wrapper">
                        <table className="admin-table">
                          <thead>
                            <tr>
                              <th>Danh mục</th>
                              <th>Số lượt tìm</th>
                              <th>Tỷ lệ Click (CTR)</th>
                            </tr>
                          </thead>
                          <tbody>
                            {categoryAnalytics.length === 0 ? (
                              <tr>
                                <td colSpan={3} style={{ textAlign: 'center', padding: '24px', color: 'var(--text-secondary)' }}>
                                  Chưa có dữ liệu danh mục sản phẩm
                                </td>
                              </tr>
                            ) : (
                              categoryAnalytics.map((cat, idx) => (
                                <tr key={idx}>
                                  <td style={{ fontWeight: 600 }}>{cat.category_name}</td>
                                  <td>{cat.search_count.toLocaleString()}</td>
                                  <td>{cat.ctr}%</td>
                                </tr>
                              ))
                            )}
                          </tbody>
                        </table>
                      </div>
                    </section>
                  </div>
                </>
              )}            </div>
          )}

          {/* TAB 2: AI SUGGESTIONS ENGINE (BACKEND PAGINATED, FILTERED, SEARCHED) */}
          {activeTab === 'ai-suggestions' && (
            <div>
              {/* Header card with action triggers */}
              <div className="admin-card" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', flexWrap: 'wrap', gap: '16px', marginBottom: '24px' }}>
                <div>
                  <h3 style={{ fontWeight: 700, fontSize: '1.1rem', marginBottom: '4px' }}>
                    Trung tâm đề xuất tối ưu hóa AI
                  </h3>
                  <p style={{ color: 'var(--text-secondary)', fontSize: '0.85rem' }}>
                    AI phân tích lịch sử tìm kiếm kém hiệu quả để đề xuất chính tả (Typo) hoặc từ đồng nghĩa (Synonym).
                  </p>
                </div>

                <div style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
                  <button 
                    onClick={handleGenerateAISuggestions}
                    disabled={generatingSuggestions || loadingTypos || loadingSynonyms}
                    className="btn btn-primary"
                    style={{ height: '36px', display: 'flex', alignItems: 'center', gap: '6px' }}
                  >
                    <Sparkles className={generatingSuggestions ? 'spinner' : ''} size={14} />
                    {generatingSuggestions ? 'Đang phân tích...' : 'Phân tích AI ngay'}
                  </button>

                  <button 
                    onClick={() => { fetchTypoSuggestions(); fetchSynonymSuggestions(); }}
                    disabled={loadingTypos || loadingSynonyms || generatingSuggestions}
                    className="btn btn-outline"
                    style={{ height: '36px', display: 'flex', alignItems: 'center', gap: '6px' }}
                  >
                    <RefreshCw className={(loadingTypos || loadingSynonyms) ? 'spinner' : ''} size={14} />
                    Tải lại
                  </button>
                </div>
              </div>

              {/* TABLE 1: TYPO CORRECTIONS */}
              <section className="admin-card" style={{ marginBottom: '24px' }}>
                <div className="table-controls">
                  <div className="admin-card-title" style={{ margin: 0 }}>
                    <AlertTriangle size={18} style={{ color: 'var(--warning)' }} />
                    <span>Đề xuất Sửa lỗi chính tả (Typo Corrections)</span>
                    <span className="badge-status" style={{ backgroundColor: 'var(--primary-light)', color: 'var(--primary)', marginLeft: '8px' }}>
                      {typoTotal} đề xuất
                    </span>
                  </div>

                  {/* Independent filter and search controls for Typo */}
                  <div style={{ display: 'flex', gap: '12px', alignItems: 'center', flexWrap: 'wrap' }}>
                    <select
                      value={typoStatus}
                      onChange={(e) => { setTypoStatus(e.target.value as any); setTypoPage(1); }}
                      className="btn btn-outline"
                      style={{ height: '38px', fontSize: '0.8rem', padding: '0 12px', borderRadius: 'var(--radius-sm)' }}
                    >
                      <option value="pending">Pending</option>
                      <option value="approved">Approved</option>
                      <option value="rejected">Rejected</option>
                    </select>

                    <div className="search-input-wrapper" style={{ margin: 0 }}>
                      <Search className="search-icon-inside" size={16} />
                      <input 
                        type="text"
                        placeholder="Tìm từ gốc hoặc gợi ý..."
                        value={typoSearch}
                        onChange={(e) => { setTypoSearch(e.target.value); setTypoPage(1); }}
                      />
                    </div>
                  </div>
                </div>

                <div className="table-wrapper" style={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0 }}>
                  <table className="admin-table">
                    <thead>
                      <tr>
                        <th>Từ gõ sai (Source)</th>
                        <th>Gợi ý thay thế (Suggested)</th>
                        <th>Độ tin cậy</th>
                        <th>Status</th>
                        <th>Hành động</th>
                      </tr>
                    </thead>
                    <tbody>
                      {typoSuggestions.length > 0 ? (
                        typoSuggestions.map((sugg) => (
                          <tr key={sugg.id}>
                            <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{sugg.source_value}</td>
                            <td style={{ fontWeight: 700, color: 'var(--primary)' }}>{sugg.suggested_value}</td>
                            <td style={{ fontWeight: 600 }}>{(sugg.confidence_score * 100).toFixed(0)}%</td>
                            <td>
                              <span className={`badge-status ${sugg.status}`} style={{
                                backgroundColor: sugg.status === 'approved' ? 'var(--success-light)' : sugg.status === 'rejected' ? '#fee2e2' : 'var(--primary-light)',
                                color: sugg.status === 'approved' ? 'var(--success)' : sugg.status === 'rejected' ? 'var(--danger)' : 'var(--primary)'
                              }}>
                                {sugg.status.toUpperCase()}
                              </span>
                            </td>
                            <td>
                              {sugg.status === 'pending' && (
                                <div style={{ display: 'flex', gap: '8px' }}>
                                  <button 
                                    onClick={() => handleApproveSuggestion(sugg.id, 'typo')}
                                    className="btn btn-primary" 
                                    style={{ padding: '4px 10px', fontSize: '0.75rem', height: '28px', backgroundColor: 'var(--success)', borderColor: 'var(--success)' }}
                                  >
                                    Duyệt
                                  </button>
                                  <button 
                                    onClick={() => handleRejectSuggestion(sugg.id, 'typo')}
                                    className="btn btn-outline" 
                                    style={{ padding: '4px 10px', fontSize: '0.75rem', height: '28px', color: 'var(--danger)', borderColor: 'var(--danger)' }}
                                  >
                                    Từ chối
                                  </button>
                                </div>
                              )}
                            </td>
                          </tr>
                        ))
                      ) : (
                        <tr>
                          <td colSpan={5} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '24px' }}>
                            {loadingTypos ? 'Đang tải đề xuất sửa lỗi...' : 'Không tìm thấy đề xuất chính tả nào.'}
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>

                {/* Typo Pagination */}
                <div className="pagination-container">
                  <div className="pagination-info">
                    Hiển thị {typoSuggestions.length > 0 ? typoStartIndex + 1 : 0} - {Math.min(typoStartIndex + itemsPerPage, typoTotal)} trong tổng số {typoTotal} đề xuất
                  </div>
                  <div className="pagination-buttons">
                    <button 
                      disabled={typoPage === 1}
                      onClick={() => setTypoPage(typoPage - 1)}
                      className="pagination-btn"
                    >
                      <ChevronLeft size={14} /> Trước
                    </button>
                    <span style={{ fontSize: '0.8rem', alignSelf: 'center', fontWeight: 600, color: 'var(--text-secondary)', padding: '0 8px' }}>
                      Trang {typoPage} / {totalTypoPages}
                    </span>
                    <button 
                      disabled={typoPage === totalTypoPages}
                      onClick={() => setTypoPage(typoPage + 1)}
                      className="pagination-btn"
                    >
                      Sau <ChevronRight size={14} />
                    </button>
                  </div>
                </div>
              </section>

              {/* TABLE 2: SYNONYM SUGGESTIONS */}
              <section className="admin-card">
                <div className="table-controls">
                  <div className="admin-card-title" style={{ margin: 0 }}>
                    <Sparkles size={18} style={{ color: 'var(--primary)' }} />
                    <span>Đề xuất Từ đồng nghĩa (Synonyms Proposals)</span>
                    <span className="badge-status" style={{ backgroundColor: 'rgba(59, 130, 246, 0.1)', color: 'var(--secondary)', marginLeft: '8px' }}>
                      {synonymTotal} đề xuất
                    </span>
                  </div>

                  {/* Independent filter and search controls for Synonym */}
                  <div style={{ display: 'flex', gap: '12px', alignItems: 'center', flexWrap: 'wrap' }}>
                    <select
                      value={synonymStatus}
                      onChange={(e) => { setSynonymStatus(e.target.value as any); setSynonymPage(1); }}
                      className="btn btn-outline"
                      style={{ height: '38px', fontSize: '0.8rem', padding: '0 12px', borderRadius: 'var(--radius-sm)' }}
                    >
                      <option value="pending">Pending</option>
                      <option value="approved">Approved</option>
                      <option value="rejected">Rejected</option>
                    </select>

                    <div className="search-input-wrapper" style={{ margin: 0 }}>
                      <Search className="search-icon-inside" size={16} />
                      <input 
                        type="text"
                        placeholder="Tìm từ gốc hoặc gợi ý..."
                        value={synonymSearch}
                        onChange={(e) => { setSynonymSearch(e.target.value); setSynonymPage(1); }}
                      />
                    </div>
                  </div>
                </div>

                <div className="table-wrapper" style={{ borderBottomLeftRadius: 0, borderBottomRightRadius: 0 }}>
                  <table className="admin-table">
                    <thead>
                      <tr>
                        <th>Từ gốc (Source)</th>
                        <th>Gợi ý đồng nghĩa (Synonym)</th>
                        <th>Độ tin cậy</th>
                        <th>Status</th>
                        <th>Hành động</th>
                      </tr>
                    </thead>
                    <tbody>
                      {synonymSuggestions.length > 0 ? (
                        synonymSuggestions.map((sugg) => (
                          <tr key={sugg.id}>
                            <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{sugg.source_value}</td>
                            <td style={{ fontWeight: 700, color: 'var(--success)' }}>{sugg.suggested_value}</td>
                            <td style={{ fontWeight: 600 }}>{(sugg.confidence_score * 100).toFixed(0)}%</td>
                            <td>
                              <span className={`badge-status ${sugg.status}`} style={{
                                backgroundColor: sugg.status === 'approved' ? 'var(--success-light)' : sugg.status === 'rejected' ? '#fee2e2' : 'var(--primary-light)',
                                color: sugg.status === 'approved' ? 'var(--success)' : sugg.status === 'rejected' ? 'var(--danger)' : 'var(--primary)'
                              }}>
                                {sugg.status.toUpperCase()}
                              </span>
                            </td>
                            <td>
                              {sugg.status === 'pending' && (
                                <div style={{ display: 'flex', gap: '8px' }}>
                                  <button 
                                    onClick={() => handleApproveSuggestion(sugg.id, 'synonym')}
                                    className="btn btn-primary" 
                                    style={{ padding: '4px 10px', fontSize: '0.75rem', height: '28px', backgroundColor: 'var(--success)', borderColor: 'var(--success)' }}
                                  >
                                    Duyệt
                                  </button>
                                  <button 
                                    onClick={() => handleRejectSuggestion(sugg.id, 'synonym')}
                                    className="btn btn-outline" 
                                    style={{ padding: '4px 10px', fontSize: '0.75rem', height: '28px', color: 'var(--danger)', borderColor: 'var(--danger)' }}
                                  >
                                    Từ chối
                                  </button>
                                </div>
                              )}
                            </td>
                          </tr>
                        ))
                      ) : (
                        <tr>
                          <td colSpan={5} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '24px' }}>
                            {loadingSynonyms ? 'Đang tải đề xuất đồng nghĩa...' : 'Không tìm thấy đề xuất đồng nghĩa nào.'}
                          </td>
                        </tr>
                      )}
                    </tbody>
                  </table>
                </div>

                {/* Synonym Pagination */}
                <div className="pagination-container">
                  <div className="pagination-info">
                    Hiển thị {synonymSuggestions.length > 0 ? synonymStartIndex + 1 : 0} - {Math.min(synonymStartIndex + itemsPerPage, synonymTotal)} trong tổng số {synonymTotal} đề xuất
                  </div>
                  <div className="pagination-buttons">
                    <button 
                      disabled={synonymPage === 1}
                      onClick={() => setSynonymPage(synonymPage - 1)}
                      className="pagination-btn"
                    >
                      <ChevronLeft size={14} /> Trước
                    </button>
                    <span style={{ fontSize: '0.8rem', alignSelf: 'center', fontWeight: 600, color: 'var(--text-secondary)', padding: '0 8px' }}>
                      Trang {synonymPage} / {totalSynonymPages}
                    </span>
                    <button 
                      disabled={synonymPage === totalSynonymPages}
                      onClick={() => setSynonymPage(synonymPage + 1)}
                      className="pagination-btn"
                    >
                      Sau <ChevronRight size={14} />
                    </button>
                  </div>
                </div>
              </section>
            </div>
          )}

          {/* TAB 3: DICTIONARIES (ACTIVE SPELLCHECK & SYNONYMS) */}
          {activeTab === 'dictionaries' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
              <section className="admin-card">
                <div className="admin-card-title" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', width: '100%', borderBottom: '1px solid var(--border-color)', paddingBottom: '16px', marginBottom: '16px' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <BookOpen size={18} style={{ color: 'var(--primary)' }} />
                    <span>Danh sách cấu hình quy tắc từ khóa (Active Rules)</span>
                  </div>
                  <button 
                    onClick={() => setShowAddRuleModal(true)} 
                    className="btn btn-primary" 
                    style={{ display: 'flex', gap: '6px', alignItems: 'center', height: '36px', fontSize: '0.85rem' }}
                  >
                    <Plus size={16} /> Thêm quy tắc mới
                  </button>
                </div>
                <p style={{ fontSize: '0.85rem', color: 'var(--text-secondary)', marginBottom: '24px' }}>
                  Đây là các luật sửa chính tả và từ đồng nghĩa đang được hệ thống Search Engine áp dụng trực tiếp khi khách hàng gõ từ khóa.
                </p>

                {/* Synonyms list */}
                <div style={{ marginBottom: '32px' }}>
                  <h3 style={{ fontSize: '0.95rem', fontWeight: 700, marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-primary)' }}>
                    <Sparkles size={16} style={{ color: 'var(--primary)' }} /> Quy tắc từ đồng nghĩa (Active Synonyms)
                  </h3>
                  <div className="table-wrapper">
                    <table className="admin-table">
                      <thead>
                        <tr>
                          <th>Từ khóa chính (Keyword)</th>
                          <th>Các từ đồng nghĩa tương đương (Synonyms)</th>
                          <th>Status</th>
                          <th style={{ width: '100px', textAlign: 'center' }}>Hành động</th>
                        </tr>
                      </thead>
                      <tbody>
                        {synonymRules.length > 0 ? (
                          synonymRules.map((rule) => (
                            <tr key={rule.id}>
                              <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{rule.keyword}</td>
                              <td style={{ fontWeight: 600, color: 'var(--success)' }}>{rule.synonym}</td>
                              <td>
                                <span className="badge-status active">ACTIVE</span>
                              </td>
                              <td style={{ textAlign: 'center' }}>
                                <button 
                                  onClick={() => handleDeleteSynonym(rule.id)}
                                  className="btn btn-outline" 
                                  style={{ padding: '4px 8px', color: 'var(--danger)', borderColor: 'rgba(239, 68, 68, 0.2)', height: '28px', fontSize: '0.75rem', fontWeight: 600 }}
                                >
                                  Xóa
                                </button>
                              </td>
                            </tr>
                          ))
                        ) : (
                          <tr>
                            <td colSpan={4} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '16px' }}>
                              {loadingDictionaries ? 'Đang tải quy tắc đồng nghĩa...' : 'Chưa có quy tắc từ đồng nghĩa nào hoạt động.'}
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>

                {/* Spellcheck list */}
                <div>
                  <h3 style={{ fontSize: '0.95rem', fontWeight: 700, marginBottom: '12px', display: 'flex', alignItems: 'center', gap: '6px', color: 'var(--text-primary)' }}>
                    <AlertTriangle size={16} style={{ color: 'var(--warning)' }} /> Quy tắc sửa lỗi chính tả (Active Spellcheck)
                  </h3>
                  <div className="table-wrapper">
                    <table className="admin-table">
                      <thead>
                        <tr>
                          <th>Từ gõ sai (Typo Word)</th>
                          <th>Từ sửa đúng (Correct Word)</th>
                          <th>Status</th>
                          <th style={{ width: '100px', textAlign: 'center' }}>Hành động</th>
                        </tr>
                      </thead>
                      <tbody>
                        {spellcheckRules.length > 0 ? (
                          spellcheckRules.map((rule) => (
                            <tr key={rule.id}>
                              <td style={{ fontWeight: 600, color: 'var(--text-primary)' }}>{rule.typo_word}</td>
                              <td style={{ fontWeight: 600, color: 'var(--primary)' }}>{rule.correct_word}</td>
                              <td>
                                <span className="badge-status active">ACTIVE</span>
                              </td>
                              <td style={{ textAlign: 'center' }}>
                                <button 
                                  onClick={() => handleDeleteSpellcheck(rule.id)}
                                  className="btn btn-outline" 
                                  style={{ padding: '4px 8px', color: 'var(--danger)', borderColor: 'rgba(239, 68, 68, 0.2)', height: '28px', fontSize: '0.75rem', fontWeight: 600 }}
                                >
                                  Xóa
                                </button>
                              </td>
                            </tr>
                          ))
                        ) : (
                          <tr>
                            <td colSpan={4} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '16px' }}>
                              {loadingDictionaries ? 'Đang tải từ điển sửa lỗi...' : 'Chưa có quy tắc sửa chính tả nào hoạt động.'}
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </div>
              </section>
            </div>
          )}

          {/* TAB 4: DATABASE SYNC & LOGS */}
          {activeTab === 'sync' && (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '24px' }}>
              <div className="admin-grid" style={{ gridTemplateColumns: '320px 1fr' }}>
                {/* Sync controls */}
                <section className="admin-card">
                  <div className="admin-card-title">
                    <RefreshCw size={18} style={{ color: 'var(--primary)' }} />
                    <span>Kích hoạt đồng bộ</span>
                  </div>
                  <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '20px' }}>
                    Đồng bộ hóa toàn bộ sản phẩm của Tenant từ Postgres sang OpenSearch index.
                  </p>

                  <button
                    disabled={syncing}
                    onClick={handleSyncDatabase}
                    className="btn btn-primary"
                    style={{ width: '100%', height: '46px', display: 'flex', alignItems: 'center', gap: '8px', justifyContent: 'center' }}
                  >
                    {syncing ? (
                      <>
                        <RefreshCw className="spinner" size={18} />
                        Đang đồng bộ...
                      </>
                    ) : (
                      <>
                        <RefreshCw size={18} />
                        Đồng bộ ngay
                      </>
                    )}
                  </button>

                  {syncMessage && (
                    <div 
                      style={{ 
                        marginTop: '16px', 
                        padding: '12px', 
                        borderRadius: '8px', 
                        fontSize: '0.8rem', 
                        display: 'flex', 
                        gap: '8px',
                        backgroundColor: syncMessage.type === 'success' ? 'var(--success-light)' : '#fee2e2',
                        color: syncMessage.type === 'success' ? 'var(--success)' : 'var(--danger)',
                        border: `1px solid ${syncMessage.type === 'success' ? 'rgba(16, 185, 129, 0.2)' : 'rgba(239, 68, 68, 0.2)'}`
                      }}
                    >
                      {syncMessage.type === 'success' ? (
                        <CheckCircle2 size={16} style={{ flexShrink: 0 }} />
                      ) : (
                        <AlertTriangle size={16} style={{ flexShrink: 0 }} />
                      )}
                      <span>{syncMessage.text}</span>
                    </div>
                  )}
                </section>

                {/* Sync Logs list */}
                <section className="admin-card">
                  <div className="admin-card-title" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                      <Clock size={18} style={{ color: 'var(--primary)' }} />
                      <span>Nhật ký lịch sử đồng bộ (Sync Logs)</span>
                    </div>
                    {syncLogs.length > 0 && (
                      <button 
                        onClick={clearSyncLogs} 
                        style={{ fontSize: '0.75rem', color: 'var(--danger)', cursor: 'pointer', fontWeight: 600, background: 'none', border: 'none' }}
                      >
                        Xóa lịch sử
                      </button>
                    )}
                  </div>
                  <p style={{ fontSize: '0.8rem', color: 'var(--text-secondary)', marginBottom: '16px' }}>
                    Lịch sử chạy đồng bộ Postgres sang OpenSearch thủ công của Admin.
                  </p>

                  <div className="table-wrapper">
                    <table className="admin-table">
                      <thead>
                        <tr>
                          <th>Thời gian chạy</th>
                          <th>Cửa hàng (Tenant)</th>
                          <th>Số lượng cập nhật</th>
                          <th>Kết quả trạng thái</th>
                        </tr>
                      </thead>
                      <tbody>
                        {syncLogs.length > 0 ? (
                          syncLogs.map((log, idx) => (
                            <tr key={idx}>
                              <td>{log.timestamp}</td>
                              <td>{log.tenantName}</td>
                              <td style={{ fontWeight: 600 }}>{log.syncedCount} SP</td>
                              <td>
                                <span 
                                  className="badge-status" 
                                  style={{ 
                                    backgroundColor: log.status === 'SUCCESS' ? 'var(--success-light)' : '#fee2e2',
                                    color: log.status === 'SUCCESS' ? 'var(--success)' : 'var(--danger)'
                                  }}
                                >
                                  {log.status === 'SUCCESS' ? 'THÀNH CÔNG' : 'THẤT BẠI'}
                                </span>
                              </td>
                            </tr>
                          ))
                        ) : (
                          <tr>
                            <td colSpan={4} style={{ textAlign: 'center', color: 'var(--text-muted)', fontStyle: 'italic', padding: '24px' }}>
                              Chưa ghi nhận lịch sử đồng bộ dữ liệu nào.
                            </td>
                          </tr>
                        )}
                      </tbody>
                    </table>
                  </div>
                </section>
              </div>
            </div>
          )}

          {/* TAB 5: AI ASSISTANT */}
          {activeTab === 'ai-assistant' && (
            <div className="assistant-chat-container">
              {/* Sidebar: Conversations Drawer */}
              <div className="assistant-sidebar">
                <div className="sidebar-header">
                  <button 
                    onClick={handleNewConversation}
                    className="btn btn-primary"
                    style={{ width: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', gap: '8px', fontSize: '0.8rem', height: '36px' }}
                  >
                    <Plus size={16} />
                    Hội thoại mới
                  </button>
                </div>
                
                <div className="conversations-scroll">
                  {loadingConversations ? (
                    <div style={{ padding: '20px', textAlign: 'center', fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                      Đang tải...
                    </div>
                  ) : conversations.length === 0 ? (
                    <div style={{ padding: '20px', textAlign: 'center', fontSize: '0.8rem', color: 'var(--text-muted)', fontStyle: 'italic' }}>
                      Chưa có hội thoại nào.
                    </div>
                  ) : (
                    conversations.map(conv => (
                      <div 
                        key={conv.id}
                        onClick={() => setActiveConversation(conv)}
                        className={`conversation-item ${activeConversation?.id === conv.id ? 'active' : ''}`}
                      >
                        <MessageSquare size={14} style={{ flexShrink: 0 }} />
                        <span className="conversation-title">{conv.title}</span>
                        <button 
                          onClick={(e) => handleDeleteConversation(e, conv.id)}
                          className="conversation-delete-btn"
                          title="Xóa hội thoại"
                        >
                          <Trash2 size={12} />
                        </button>
                      </div>
                    ))
                  )}
                </div>
              </div>

              {/* Main chat column */}
              <div className="assistant-main-chat">
                {/* Chat Header */}
                <div className="chat-header">
                  <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                    <div style={{ backgroundColor: 'var(--primary-light)', color: 'var(--primary)', padding: '8px', borderRadius: '50%', display: 'flex' }}>
                      <Bot size={18} />
                    </div>
                    <div>
                      <h3 style={{ fontSize: '0.95rem', fontWeight: 700, color: 'var(--text-primary)' }}>
                        {activeConversation ? activeConversation.title : 'Trợ lý AI SwiftSearch'}
                      </h3>
                      <p style={{ fontSize: '0.75rem', color: 'var(--text-secondary)' }}>
                        Tra cứu sản phẩm và tạo đề xuất cấu hình từ điển bằng tiếng Việt.
                      </p>
                    </div>
                  </div>
                </div>

                {/* Chat Message Box */}
                <div className="chat-messages-list">
                  {chatMessages.map((msg, msgIdx) => (
                    <div 
                      key={msgIdx} 
                      className={`chat-message-bubble ${msg.role}`}
                    >
                      {/* Render message content */}
                      <div style={{ whiteSpace: 'pre-wrap' }}>
                        {msg.content}
                      </div>

                      {/* Render proposed actions if assistant bubble contains any */}
                      {msg.role === 'assistant' && msg.proposed_actions && msg.proposed_actions.length > 0 && (
                        <div style={{ display: 'flex', flexDirection: 'column', gap: '8px', marginTop: '4px', width: '100%' }}>
                          {msg.proposed_actions.map((act, actIdx) => {
                            const actionState = msg.action_states?.[String(actIdx)] || 'pending';
                            return (
                              <div key={actIdx} className="proposed-action-card">
                                <div className="proposed-action-header">
                                  <Sparkles size={14} />
                                  <span>ĐỀ XUẤT CỦA AI TRỢ LÝ</span>
                                </div>
                                <div className="proposed-action-desc">
                                  {act.description}
                                </div>
                                <div className="proposed-action-details">
                                  {JSON.stringify(act.params, null, 2)}
                                </div>
                                <div className="proposed-action-actions">
                                  {actionState === 'pending' ? (
                                    <>
                                      <button 
                                        onClick={() => handleApproveAction(msgIdx, actIdx, act)}
                                        className="action-btn-approve"
                                      >
                                        Chấp nhận
                                      </button>
                                      <button 
                                        onClick={() => handleRejectAction(msgIdx, actIdx, act)}
                                        className="action-btn-reject"
                                      >
                                        Từ chối
                                      </button>
                                    </>
                                  ) : (
                                    <div className={`action-card-status ${actionState}`}>
                                      {actionState === 'accepted' ? '✓ ĐÃ ÁP DỤNG THÀNH CÔNG' : '✗ ĐÃ TỪ CHỐI'}
                                    </div>
                                  )}
                                </div>
                              </div>
                            );
                          })}
                        </div>
                      )}
                    </div>
                  ))}

                  {/* Typing Indicator */}
                  {sendingChat && (
                    <div className="chat-message-bubble assistant">
                      <div className="assistant-typing">
                        <div className="typing-dot" />
                        <div className="typing-dot" />
                        <div className="typing-dot" />
                      </div>
                    </div>
                  )}
                </div>

                {/* Chat Input form and Quick prompts */}
                <div className="chat-input-area">
                  <div className="quick-prompts-container">
                    <button 
                      onClick={() => handleSendChat(undefined, "Bàn phím cơ Akko còn bao nhiêu cái trong kho?")}
                      className="quick-prompt-btn"
                    >
                      🔍 Check bàn phím Akko
                    </button>
                    <button 
                      onClick={() => handleSendChat(undefined, "Tìm các sản phẩm thương hiệu Logitech")}
                      className="quick-prompt-btn"
                    >
                      🔍 Check sản phẩm Logitech
                    </button>
                    <button 
                      onClick={() => handleSendChat(undefined, "Thêm từ đồng nghĩa của bàn phím cơ là phím cơ")}
                      className="quick-prompt-btn"
                    >
                      ➕ Thêm từ đồng nghĩa: bàn phím cơ {"<->"} phím cơ
                    </button>
                    <button 
                      onClick={() => handleSendChat(undefined, "Tạo luật sửa lỗi chính tả gõ sai ipone thành iPhone")}
                      className="quick-prompt-btn"
                    >
                      ✏️ Sửa chính tả: ipone {"->"} iPhone
                    </button>
                  </div>
                  
                  <form onSubmit={handleSendChat} className="chat-input-form">
                    <input 
                      type="text"
                      placeholder="Gõ tin nhắn hỏi đáp sản phẩm hoặc yêu cầu cấu hình từ điển..."
                      value={chatInput}
                      onChange={(e) => setChatInput(e.target.value)}
                      disabled={sendingChat}
                      className="chat-input-field"
                    />
                    <button 
                      type="submit" 
                      disabled={sendingChat || !chatInput.trim()}
                      className="btn btn-primary"
                      style={{ height: '42px', padding: '0 24px', fontWeight: 600 }}
                    >
                      Gửi
                    </button>
                  </form>
                </div>
              </div>
            </div>
          )}
        </main>

        <Footer />
      </div>

      {showAddRuleModal && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: 'rgba(0, 0, 0, 0.5)',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          zIndex: 1000,
          backdropFilter: 'blur(4px)'
        }}>
          <div className="admin-card" style={{
            width: '100%',
            maxWidth: '500px',
            margin: '20px',
            position: 'relative',
            boxShadow: 'var(--shadow-lg)'
          }}>
            <button 
              onClick={() => setShowAddRuleModal(false)}
              style={{
                position: 'absolute',
                top: '16px',
                right: '16px',
                background: 'none',
                border: 'none',
                cursor: 'pointer',
                color: 'var(--text-secondary)'
              }}
            >
              <X size={20} />
            </button>

            <div className="admin-card-title" style={{ marginBottom: '20px' }}>
              <Plus size={18} style={{ color: 'var(--primary)' }} />
              <span>Thêm Quy tắc Từ điển Thủ công</span>
            </div>

            <form onSubmit={handleAddRule}>
              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                  Loại quy tắc
                </label>
                <div style={{ display: 'flex', gap: '16px' }}>
                  <label style={{ display: 'flex', gap: '8px', alignItems: 'center', fontSize: '0.85rem', cursor: 'pointer' }}>
                    <input 
                      type="radio" 
                      name="rule_type" 
                      checked={newRuleType === 'synonym'} 
                      onChange={() => setNewRuleType('synonym')} 
                    />
                    Từ đồng nghĩa (Synonym)
                  </label>
                  <label style={{ display: 'flex', gap: '8px', alignItems: 'center', fontSize: '0.85rem', cursor: 'pointer' }}>
                    <input 
                      type="radio" 
                      name="rule_type" 
                      checked={newRuleType === 'spellcheck'} 
                      onChange={() => setNewRuleType('spellcheck')} 
                    />
                    Sửa lỗi chính tả (Spellcheck)
                  </label>
                </div>
              </div>

              {newRuleType === 'synonym' ? (
                <>
                  <div style={{ marginBottom: '16px' }}>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                      Từ khóa chính (Keyword)
                    </label>
                    <input 
                      type="text" 
                      value={newKeyword}
                      onChange={(e) => setNewKeyword(e.target.value)}
                      placeholder="Ví dụ: keyboard"
                      style={{
                        width: '100%',
                        height: '40px',
                        padding: '0 12px',
                        borderRadius: 'var(--radius-sm)',
                        border: '1.5px solid var(--border-color)',
                        fontSize: '0.85rem',
                        backgroundColor: 'var(--bg-primary)',
                        color: 'var(--text-primary)'
                      }}
                      required
                    />
                  </div>
                  <div style={{ marginBottom: '16px' }}>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                      Từ đồng nghĩa (Synonym)
                    </label>
                    <input 
                      type="text" 
                      value={newSynonym}
                      onChange={(e) => setNewSynonym(e.target.value)}
                      placeholder="Ví dụ: bàn phím"
                      style={{
                        width: '100%',
                        height: '40px',
                        padding: '0 12px',
                        borderRadius: 'var(--radius-sm)',
                        border: '1.5px solid var(--border-color)',
                        fontSize: '0.85rem',
                        backgroundColor: 'var(--bg-primary)',
                        color: 'var(--text-primary)'
                      }}
                      required
                    />
                  </div>
                  <div style={{ marginBottom: '20px' }}>
                    <label style={{ display: 'flex', gap: '8px', alignItems: 'center', fontSize: '0.85rem', cursor: 'pointer', fontWeight: 500, color: 'var(--text-secondary)' }}>
                      <input 
                        type="checkbox" 
                        checked={isBidirectional}
                        onChange={(e) => setIsBidirectional(e.target.checked)}
                      />
                      Áp dụng quan hệ hai chiều (Bidirectional)
                    </label>
                    <span style={{ display: 'block', fontSize: '0.75rem', color: 'var(--text-muted)', marginTop: '4px', marginLeft: '22px' }}>
                      (Hệ thống tự động sinh quy tắc ngược lại: A &rarr; B và B &rarr; A)
                    </span>
                  </div>
                </>
              ) : (
                <>
                  <div style={{ marginBottom: '16px' }}>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                      Từ gõ sai (Typo Word)
                    </label>
                    <input 
                      type="text" 
                      value={newTypoWord}
                      onChange={(e) => setNewTypoWord(e.target.value)}
                      placeholder="Ví dụ: ban phin"
                      style={{
                        width: '100%',
                        height: '40px',
                        padding: '0 12px',
                        borderRadius: 'var(--radius-sm)',
                        border: '1.5px solid var(--border-color)',
                        fontSize: '0.85rem',
                        backgroundColor: 'var(--bg-primary)',
                        color: 'var(--text-primary)'
                      }}
                      required
                    />
                  </div>
                  <div style={{ marginBottom: '20px' }}>
                    <label style={{ display: 'block', fontSize: '0.85rem', fontWeight: 600, marginBottom: '6px', color: 'var(--text-secondary)' }}>
                      Từ sửa đúng (Correct Word)
                    </label>
                    <input 
                      type="text" 
                      value={newCorrectWord}
                      onChange={(e) => setNewCorrectWord(e.target.value)}
                      placeholder="Ví dụ: bàn phím"
                      style={{
                        width: '100%',
                        height: '40px',
                        padding: '0 12px',
                        borderRadius: 'var(--radius-sm)',
                        border: '1.5px solid var(--border-color)',
                        fontSize: '0.85rem',
                        backgroundColor: 'var(--bg-primary)',
                        color: 'var(--text-primary)'
                      }}
                      required
                    />
                  </div>
                </>
              )}

              <div style={{ display: 'flex', gap: '12px', justifyContent: 'flex-end' }}>
                <button 
                  type="button" 
                  onClick={() => setShowAddRuleModal(false)} 
                  className="btn btn-outline"
                  style={{ height: '38px', fontSize: '0.85rem' }}
                >
                  Hủy
                </button>
                <button 
                  type="submit" 
                  disabled={actionLoading}
                  className="btn btn-primary"
                  style={{ height: '38px', fontSize: '0.85rem', minWidth: '100px' }}
                >
                  {actionLoading ? 'Đang tạo...' : 'Lưu quy tắc'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};

export default Admin;
