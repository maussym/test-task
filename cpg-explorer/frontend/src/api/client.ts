import axios from 'axios';

const API_BASE_URL = import.meta.env.VITE_API_URL || 'http://localhost:8080/api';

const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
});

export interface Node {
  id: string;
  kind: string;
  name: string;
  file?: string;
  line?: number;
  col?: number;
  end_line?: number;
  package?: string;
  parent_function?: string;
  type_info?: string;
  properties?: Record<string, any>;
}

export interface Edge {
  source: string;
  target: string;
  kind: string;
  properties?: Record<string, any>;
}

export interface Graph {
  nodes: Node[];
  edges: Edge[];
}

export interface Function {
  id: string;
  name: string;
  package?: string;
  file?: string;
  line?: number;
  cyclomatic_complexity?: number;
  fan_in?: number;
  fan_out?: number;
  loc?: number;
  num_params?: number;
}

export interface Package {
  name: string;
  file_count: number;
  function_count: number;
  total_loc: number;
  total_complexity: number;
  avg_complexity: number;
  max_complexity: number;
  type_count: number;
  interface_count: number;
}

export interface Stats {
  total_nodes: number;
  total_edges: number;
  total_files: number;
  total_packages: number;
  total_functions: number;
  total_types: number;
  total_metrics: number;
}

export const api = {
  getHealth: async () => {
    const response = await apiClient.get('/health');
    return response.data;
  },

  getStats: async (): Promise<Stats> => {
    const response = await apiClient.get<Stats>('/stats');
    return response.data;
  },

  searchSymbols: async (query: string, kind?: string, limit: number = 50): Promise<Node[]> => {
    const response = await apiClient.get<Node[]>('/search', {
      params: { q: query, kind, limit },
    });
    return response.data;
  },

  getFunction: async (id: string): Promise<Function> => {
    const response = await apiClient.get<Function>(`/functions/${encodeURIComponent(id)}`);
    return response.data;
  },

  getCallGraph: async (id: string, depth: number = 2): Promise<Graph> => {
    const response = await apiClient.get<Graph>(`/functions/${encodeURIComponent(id)}/callgraph`, {
      params: { depth },
    });
    return response.data;
  },

  getPackages: async (): Promise<Package[]> => {
    const response = await apiClient.get<Package[]>('/packages');
    return response.data;
  },

  getPackageGraph: async (): Promise<Graph> => {
    const response = await apiClient.get<Graph>('/packages/graph');
    return response.data;
  },

  getPackageFunctions: async (packageName: string, limit: number = 20): Promise<Function[]> => {
    const response = await apiClient.get<Function[]>(`/packages/${packageName}/functions`, {
      params: { limit },
    });
    return response.data;
  },

  getSourceCode: async (file: string): Promise<{ file: string; content: string }> => {
    const response = await apiClient.get('/source', {
      params: { file },
    });
    return response.data;
  },
};
