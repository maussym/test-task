import { useState } from 'react';
import { QueryClient, QueryClientProvider, useQuery } from '@tanstack/react-query';
import { api } from './api/client';
import { GraphViewer } from './components/GraphViewer';

const queryClient = new QueryClient();

function AppContent() {
  const [selectedFunctionId, setSelectedFunctionId] = useState<string | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [view, setView] = useState<'call-graph' | 'package-graph'>('call-graph');

  // Fetch stats
  const { data: stats } = useQuery({
    queryKey: ['stats'],
    queryFn: api.getStats,
  });

  // Search symbols
  const { data: searchResults, refetch: searchRefetch } = useQuery({
    queryKey: ['search', searchQuery],
    queryFn: () => api.searchSymbols(searchQuery, 'function', 20),
    enabled: searchQuery.length > 2,
  });

  // Fetch call graph
  const { data: callGraph, isLoading: callGraphLoading } = useQuery({
    queryKey: ['callgraph', selectedFunctionId],
    queryFn: () => api.getCallGraph(selectedFunctionId!, 2),
    enabled: !!selectedFunctionId && view === 'call-graph',
  });

  // Fetch package graph
  const { data: packageGraph, isLoading: packageGraphLoading } = useQuery({
    queryKey: ['packagegraph'],
    queryFn: api.getPackageGraph,
    enabled: view === 'package-graph',
  });

  // Fetch function details
  const { data: functionDetails } = useQuery({
    queryKey: ['function', selectedFunctionId],
    queryFn: () => api.getFunction(selectedFunctionId!),
    enabled: !!selectedFunctionId,
  });

  const handleNodeClick = (nodeId: string) => {
    if (nodeId.startsWith('pkg::')) {
      return;
    }
    setSelectedFunctionId(nodeId);
  };

  const handleSearch = (e: React.FormEvent) => {
    e.preventDefault();
    if (searchQuery.length > 2) {
      searchRefetch();
    }
  };

  return (
    <div className="flex flex-col h-screen bg-gray-100">
      <header className="bg-white shadow-sm border-b border-gray-200 p-4">
        <div className="max-w-7xl mx-auto flex items-center justify-between">
          <div className="flex items-center space-x-4">
            <h1 className="text-2xl font-bold text-gray-900">CPG Explorer</h1>
            {stats && (
              <div className="text-sm text-gray-600">
                {stats.total_functions.toLocaleString()} functions |{' '}
                {stats.total_packages} packages |{' '}
                {stats.total_edges.toLocaleString()} edges
              </div>
            )}
          </div>
          <div className="flex space-x-2">
            <button
              onClick={() => setView('call-graph')}
              className={`px-4 py-2 rounded-md ${
                view === 'call-graph'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
              }`}
            >
              Call Graph
            </button>
            <button
              onClick={() => setView('package-graph')}
              className={`px-4 py-2 rounded-md ${
                view === 'package-graph'
                  ? 'bg-blue-600 text-white'
                  : 'bg-gray-200 text-gray-700 hover:bg-gray-300'
              }`}
            >
              Package Graph
            </button>
          </div>
        </div>
      </header>

      <div className="flex-1 flex overflow-hidden">
        <aside className="w-80 bg-white border-r border-gray-200 overflow-y-auto p-4">
          <div className="space-y-4">
            <div>
              <form onSubmit={handleSearch} className="space-y-2">
                <input
                  type="text"
                  placeholder="Search functions..."
                  value={searchQuery}
                  onChange={(e) => setSearchQuery(e.target.value)}
                  className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                />
                <button
                  type="submit"
                  className="w-full px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                  Search
                </button>
              </form>
            </div>

            {searchResults && searchResults.length > 0 && (
              <div className="space-y-2">
                <h3 className="text-sm font-semibold text-gray-700">Results</h3>
                <div className="space-y-1 max-h-96 overflow-y-auto">
                  {searchResults.map((result) => (
                    <button
                      key={result.id}
                      onClick={() => {
                        setSelectedFunctionId(result.id);
                        setView('call-graph');
                      }}
                      className="w-full text-left px-3 py-2 text-sm bg-gray-50 hover:bg-gray-100 rounded-md border border-gray-200"
                    >
                      <div className="font-medium text-gray-900">{result.name}</div>
                      <div className="text-xs text-gray-500">{result.package}</div>
                    </button>
                  ))}
                </div>
              </div>
            )}

            {functionDetails && (
              <div className="mt-4 p-4 bg-blue-50 rounded-lg border border-blue-200">
                <h3 className="text-sm font-semibold text-blue-900 mb-2">Selected Function</h3>
                <div className="space-y-1 text-sm">
                  <div className="font-medium text-blue-900">{functionDetails.name}</div>
                  <div className="text-blue-700">{functionDetails.package}</div>
                  {functionDetails.file && (
                    <div className="text-blue-600 text-xs">{functionDetails.file}:{functionDetails.line}</div>
                  )}
                  {functionDetails.cyclomatic_complexity !== undefined && (
                    <div className="mt-2 space-y-1">
                      <div className="text-blue-700">
                        Complexity: <span className="font-medium">{functionDetails.cyclomatic_complexity}</span>
                      </div>
                      <div className="text-blue-700">
                        LOC: <span className="font-medium">{functionDetails.loc}</span>
                      </div>
                      <div className="text-blue-700">
                        Fan-in: <span className="font-medium">{functionDetails.fan_in}</span> |
                        Fan-out: <span className="font-medium">{functionDetails.fan_out}</span>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}

            <div className="mt-4 p-4 bg-gray-50 rounded-lg border border-gray-200">
              <h3 className="text-sm font-semibold text-gray-700 mb-2">Instructions</h3>
              <ol className="text-xs text-gray-600 space-y-1 list-decimal list-inside">
                <li>Search for a function by name</li>
                <li>Click a result to visualize its call graph</li>
                <li>Click nodes in the graph to navigate</li>
                <li>Switch to Package Graph to see architecture</li>
              </ol>
            </div>
          </div>
        </aside>

        <main className="flex-1 p-4">
          {view === 'call-graph' && (
            <>
              {!selectedFunctionId && (
                <div className="h-full flex items-center justify-center text-gray-500">
                  <div className="text-center">
                    <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
                    </svg>
                    <h3 className="mt-2 text-sm font-medium text-gray-900">No function selected</h3>
                    <p className="mt-1 text-sm text-gray-500">Search for a function to visualize its call graph</p>
                  </div>
                </div>
              )}
              {callGraphLoading && (
                <div className="h-full flex items-center justify-center">
                  <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-600 mx-auto"></div>
                    <p className="mt-2 text-sm text-gray-600">Loading call graph...</p>
                  </div>
                </div>
              )}
              {callGraph && !callGraphLoading && (
                <GraphViewer graph={callGraph} onNodeClick={handleNodeClick} layout="dagre" />
              )}
            </>
          )}

          {view === 'package-graph' && (
            <>
              {packageGraphLoading && (
                <div className="h-full flex items-center justify-center">
                  <div className="text-center">
                    <div className="animate-spin rounded-full h-12 w-12 border-b-2 border-green-600 mx-auto"></div>
                    <p className="mt-2 text-sm text-gray-600">Loading package graph...</p>
                  </div>
                </div>
              )}
              {packageGraph && !packageGraphLoading && (
                <GraphViewer graph={packageGraph} onNodeClick={handleNodeClick} layout="cose" />
              )}
            </>
          )}
        </main>
      </div>
    </div>
  );
}

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AppContent />
    </QueryClientProvider>
  );
}

export default App;
