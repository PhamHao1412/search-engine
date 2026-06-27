import { SearchResponse, SyncResponse, ClickTrackRequest, SearchDebugInfo } from '../types';

const BASE_URL = '/api/v1';

export const searchApi = {
  /**
   * Search products by query, page, page_size
   */
  async searchProducts(
    query: string,
    page: number,
    pageSize: number,
    tenantId: string
  ): Promise<SearchResponse> {
    const params = new URLSearchParams({
      q: query,
      page: String(page),
      page_size: String(pageSize),
    });

    const response = await fetch(`${BASE_URL}/search?${params.toString()}`, {
      method: 'GET',
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
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
      headers: {
        'Accept': 'application/json',
        'X-Tenant-ID': tenantId,
      },
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
    const headers = {
      'Content-Type': 'application/json',
      'X-Tenant-ID': tenantId,
    };

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
