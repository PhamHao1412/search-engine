import { SearchResponse, SyncResponse, ClickTrackRequest, SearchDebugInfo, Suggestion, Product } from '../types';

const BASE_URL = '/api/v1';

const getHeaders = (tenantId: string, additional: Record<string, string> = {}): Record<string, string> => {
  const lang = localStorage.getItem('swiftsearch_search_lang') || 'vi';
  return {
    'Accept': 'application/json',
    'X-Tenant-ID': tenantId,
    'X-Language-Key': lang,
    ...additional,
  };
};

export const searchApi = {
  /**
   * Search products by query, page, page_size
   */
  async searchProducts(
    query: string,
    page: number,
    pageSize: number,
    tenantId: string,
    lang?: string
  ): Promise<SearchResponse> {
    const params = new URLSearchParams({
      q: query,
      page: String(page),
      page_size: String(pageSize),
    });

    const response = await fetch(`${BASE_URL}/search?${params.toString()}`, {
      method: 'GET',
      headers: getHeaders(tenantId, lang ? { 'X-Language-Key': lang } : {}),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `Search request failed with status ${response.status}`);
    }

    return response.json();
  },

  /**
   * Force sync all database products to OpenSearch index for a tenant
   */
  async syncDatabase(tenantId: string): Promise<SyncResponse> {
    const response = await fetch(`${BASE_URL}/search/sync`, {
      method: 'POST',
      headers: getHeaders(tenantId),
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error || `Sync request failed with status ${response.status}`);
    }

    return response.json();
  },

  /**
   * Log analytics click event asynchronously and non-blocking
   */
  trackClick(payload: ClickTrackRequest, tenantId: string): void {
    const url = `${BASE_URL}/analytics/click`;
    const headers = getHeaders(tenantId, { 'Content-Type': 'application/json' });

    // Use keepalive: true to ensure the request is not aborted if the page unloads
    fetch(url, {
      method: 'POST',
      headers,
      body: JSON.stringify(payload),
      keepalive: true,
    }).catch((err) => {
      console.warn('Analytics Click Tracking failed background execution:', err);
    });
  },

  /**
   * Fetch autocomplete suggestions for a given query prefix
   */
  async getSuggestions(query: string, tenantId: string): Promise<Suggestion[]> {
    if (!query || query.trim().length < 2) {
      return [];
    }
    try {
      const params = new URLSearchParams({ q: query.trim() });
      const response = await fetch(`${BASE_URL}/suggest?${params.toString()}`, {
        method: 'GET',
        headers: getHeaders(tenantId),
      });

      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.suggestions || [];
    } catch (err) {
      console.warn('Failed to fetch suggestions:', err);
      return [];
    }
  },

  /**
   * Fetch product details by ID
   */
  async getProduct(id: string, tenantId: string): Promise<Product> {
    const response = await fetch(`${BASE_URL}/products/${id}`, {
      method: 'GET',
      headers: getHeaders(tenantId),
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch product details with status ${response.status}`);
    }

    return response.json();
  },

  /**
   * Fetch dynamic hot keywords based on real search queries
   */
  async getHotKeywords(tenantId: string): Promise<string[]> {
    try {
      const response = await fetch(`${BASE_URL}/search/hot-keywords`, {
        method: 'GET',
        headers: getHeaders(tenantId),
      });

      if (!response.ok) {
        return [];
      }

      const data = await response.json();
      return data.keywords || [];
    } catch (err) {
      console.warn('Failed to fetch hot keywords:', err);
      return [];
    }
  },
};

// Client-side Search Debug Helper (Mocks spellcheck, synonyms, translations since backend APIs for these are under construction)
const SPELLCHECK_DICT: Record<string, string> = {
  'ako': 'akko',
  'akkoo': 'akko',
  'logitek': 'logitech',
  'logi': 'logitech',
  'chuot': 'chuột',
  'ban phim': 'bàn phím',
  'phim co': 'phím cơ',
  'son duong': 'son dưỡng',
  'my pham': 'mỹ phẩm',
};

const SYNONYMS_DICT: Record<string, string[]> = {
  'akko': ['bàn phím', 'keyboard', 'keycap', 'phím cơ'],
  'logitech': ['chuột', 'mouse', 'bàn di chuột', 'gaming gear'],
  'bàn phím': ['keyboard', 'phím cơ', 'keycap', 'gaming keyboard'],
  'chuột': ['mouse', 'chuột không dây', 'gaming mouse'],
  'son': ['lipstick', 'son dưỡng', 'mỹ phẩm', 'lip balm'],
  'mỹ phẩm': ['cosmetics', 'son môi', 'kem dưỡng da', 'skincare'],
};

const WORD_TRANSLATIONS: Record<string, { en: string; th: string }> = {
  'bàn phím': { en: 'keyboard', th: 'คีย์บอร์ด' },
  'phím': { en: 'key', th: 'กุญแจ' },
  'cơ': { en: 'mechanical', th: 'เครื่องกล' },
  'không': { en: 'wireless', th: 'ไร้สาย' },
  'dây': { en: 'wire', th: 'สาย' },
  'chuột': { en: 'mouse', th: 'เมาส์' },
  'son': { en: 'lipstick', th: 'ลิปสติก' },
  'dưỡng': { en: 'balm/care', th: 'บำรุง' },
  'môi': { en: 'lip', th: 'ริมฝีปาก' },
  'mỹ': { en: 'cosmetics', th: 'เครื่องสำอาง' },
  'phẩm': { en: 'product', th: 'ผลิตภัณฑ์' },
  'akko': { en: 'Akko', th: 'Akko' },
  'logitech': { en: 'Logitech', th: 'Logitech' },
};

export const getSearchDebugInfo = (
  query: string,
  searchTimeMs: number,
  isCached: boolean
): SearchDebugInfo => {
  const cleanQuery = query.trim().toLowerCase();
  
  // 1. Original & Normalized Query
  const normalized = cleanQuery.replace(/\s+/g, ' ');

  // 2. Spellcheck Simulation
  let spellcheck = '';
  const words = normalized.split(' ');
  const correctedWords = words.map(w => SPELLCHECK_DICT[w] || w);
  const correctedQuery = correctedWords.join(' ');
  if (correctedQuery !== normalized) {
    spellcheck = correctedQuery;
  }

  // 3. Synonym Simulation
  const queryTerms = spellcheck ? correctedWords : words;
  const synonymsSet = new Set<string>();
  queryTerms.forEach(term => {
    const matches = SYNONYMS_DICT[term];
    if (matches) {
      matches.forEach(m => synonymsSet.add(m));
    }
  });
  const synonyms = Array.from(synonymsSet);

  // 4. Translation Simulation
  // Simple word-by-word mapping or phrase mapping
  const phrase = spellcheck || normalized;
  let enTranslation = '';
  let thTranslation = '';

  if (phrase.includes('bàn phím cơ akko')) {
    enTranslation = 'Akko mechanical keyboard';
    thTranslation = 'คีย์บอร์ดเชิงกล Akko';
  } else if (phrase.includes('chuột logitech')) {
    enTranslation = 'Logitech mouse';
    thTranslation = 'เมาส์ Logitech';
  } else if (phrase.includes('son dưỡng môi')) {
    enTranslation = 'Moisturizing lip balm';
    thTranslation = 'ลิปบาล์มบำรุงริมฝีปาก';
  } else {
    // Word-by-word translation fallback
    const transWordsEn = queryTerms.map(w => WORD_TRANSLATIONS[w]?.en || w);
    const transWordsTh = queryTerms.map(w => WORD_TRANSLATIONS[w]?.th || w);
    enTranslation = transWordsEn.join(' ');
    thTranslation = transWordsTh.join(' ');
  }

  return {
    originalQuery: query,
    normalized,
    spellcheck,
    synonyms,
    translations: {
      vi: phrase,
      en: enTranslation || 'N/A',
      th: thTranslation || 'N/A',
    },
    cacheStatus: isCached ? 'HIT' : 'MISS',
    searchTime: searchTimeMs,
  };
};

export interface AISuggestion {
  id: string;
  tenant_id: string;
  suggestion_type: 'typo' | 'synonym';
  source_value: string;
  suggested_value: string;
  confidence_score: number;
  status: 'pending' | 'approved' | 'rejected';
  reason?: string;
  created_at: string;
}

export const adminApi = {
  async getTenants(): Promise<Array<{ id: string; name: string }>> {
    const response = await fetch(`${BASE_URL}/admin/tenants`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
      },
    });
    if (!response.ok) {
      throw new Error(`Failed to fetch tenants: ${response.status}`);
    }
    const data = await response.json();
    return data.tenants || [];
  },

  /**
   * Fetch AI suggestions for search optimization
   */
  async getAISuggestions(
    tenantId: string,
    status = 'pending',
    type = '',
    search = '',
    page = 1,
    pageSize = 5
  ): Promise<{ suggestions: AISuggestion[]; total: number }> {
    const params = new URLSearchParams({
      status,
      page: String(page),
      page_size: String(pageSize),
    });
    if (type) {
      params.append('type', type);
    }
    if (search) {
      params.append('search', search);
    }
    const response = await fetch(`${BASE_URL}/admin/ai/suggestions?${params.toString()}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch AI suggestions: ${response.status}`);
    }

    const data = await response.json();
    return {
      suggestions: data.suggestions || [],
      total: data.total || 0,
    };
  },

  /**
   * Approve an AI suggestion
   */
  async approveAISuggestion(id: string, tenantId: string): Promise<void> {
    const response = await fetch(`${BASE_URL}/admin/ai/suggestions/${id}/approve`, {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to approve AI suggestion: ${response.status}`);
    }
  },

  /**
   * Reject an AI suggestion
   */
  async rejectAISuggestion(id: string, tenantId: string): Promise<void> {
    const response = await fetch(`${BASE_URL}/admin/ai/suggestions/${id}/reject`, {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to reject AI suggestion: ${response.status}`);
    }
  },

  /**
   * Manually trigger AI analysis and suggestions generation
   */
  async generateAISuggestions(tenantId: string): Promise<void> {
    const response = await fetch(`${BASE_URL}/admin/ai/suggestions/generate`, {
      method: 'POST',
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to generate AI suggestions: ${response.status}`);
    }
  },

  /**
   * Fetch active spellcheck rules for a tenant
   */
  async getSpellcheckRules(tenantId: string): Promise<Array<{ id: string; tenant_id: string; typo_word: string; correct_word: string; status: string }>> {
    const response = await fetch(`${BASE_URL}/admin/dictionaries/spellcheck`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch spellcheck rules: ${response.status}`);
    }

    const data = await response.json();
    return data.rules || [];
  },

  /**
   * Fetch active synonym rules for a tenant
   */
  async getSearchSynonyms(tenantId: string): Promise<Array<{ id: string; tenant_id: string; keyword: string; synonym: string; status: string }>> {
    const response = await fetch(`${BASE_URL}/admin/dictionaries/synonyms`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to fetch synonym rules: ${response.status}`);
    }

    const data = await response.json();
    return data.rules || [];
  },

  async addSynonym(tenantId: string, keyword: string, synonym: string, isBidirectional: boolean): Promise<void> {
    const response = await fetch(`${BASE_URL}/admin/dictionaries/synonyms`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
      body: JSON.stringify({ keyword, synonym, is_bidirectional: isBidirectional }),
    });

    if (!response.ok) {
      throw new Error(`Failed to add synonym rule: ${response.status}`);
    }
  },

  async deleteSynonym(tenantId: string, id: string): Promise<void> {
    const response = await fetch(`${BASE_URL}/admin/dictionaries/synonyms/${id}`, {
      method: 'DELETE',
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to delete synonym rule: ${response.status}`);
    }
  },

  async addSpellcheck(tenantId: string, typoWord: string, correctWord: string): Promise<void> {
    const response = await fetch(`${BASE_URL}/admin/dictionaries/spellcheck`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
      body: JSON.stringify({ typo_word: typoWord, correct_word: correctWord }),
    });

    if (!response.ok) {
      throw new Error(`Failed to add spellcheck rule: ${response.status}`);
    }
  },

  async deleteSpellcheck(tenantId: string, id: string): Promise<void> {
    const response = await fetch(`${BASE_URL}/admin/dictionaries/spellcheck/${id}`, {
      method: 'DELETE',
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
    });

    if (!response.ok) {
      throw new Error(`Failed to delete spellcheck rule: ${response.status}`);
    }
  },
};
