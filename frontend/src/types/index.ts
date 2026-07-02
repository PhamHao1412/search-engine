export interface Product {
  id: string;
  tenant_id: string;
  product_name_vi: string;
  product_name_en?: string;
  product_name_th?: string;
  description_vi?: string;
  description_en?: string;
  description_th?: string;
  brand?: string;
  price: number;
  inventory: number;
  featured: boolean;
  status: string;
  search_tags?: string;
  image_url?: string; // May be added by user later
}

export interface SearchResponse {
  search_log_id: string;
  products: Product[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
  spellcheck_corrected?: string;
  auto_corrected?: boolean;
}

export interface SyncResponse {
  message: string;
  tenant_id: string;
  synced_count: number;
}

export interface ClickTrackRequest {
  search_log_id: string;
  product_id: string;
  query: string;
  position: number;
}

export interface SearchDebugInfo {
  originalQuery: string;
  normalized: string;
  spellcheck: string;
  synonyms: string[];
  translations: {
    vi: string;
    en: string;
    th: string;
  };
  cacheStatus: 'HIT' | 'MISS';
  searchTime: number;
}

export interface Suggestion {
  id: string;
  text: string;
  brand: string;
  price: number;
  product_name_vi: string;
  product_name_en?: string;
  product_name_th?: string;
  description_vi?: string;
  description_en?: string;
  description_th?: string;
  image_url?: string;
  inventory: number;
}
