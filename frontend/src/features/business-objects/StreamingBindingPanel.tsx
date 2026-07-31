import React, { useState } from 'react';

interface StreamingConfig {
  topicName: string;
  windowType: 'TUMBLE' | 'HOP' | 'SESSION';
  windowInterval: string;
  watermarkDelay: string;
}

export const StreamingBindingPanel: React.FC = () => {
  const [config, setConfig] = useState<StreamingConfig>({
    topicName: 'redpanda.market_ticks.live',
    windowType: 'TUMBLE',
    windowInterval: '5 MINUTE',
    watermarkDelay: '5 SECOND',
  });

  // Generate a live preview of the Flink SQL the compiler will create
  const generatePreviewSQL = () => {
    return `SELECT window_start, window_end, \n  JSON_VALUE(payload, '$.trade_amount') AS trade_amount\nFROM TABLE(\n  ${config.windowType}(TABLE ${config.topicName}, DESCRIPTOR(proctime), INTERVAL '${config.windowInterval}')\n)\nWHERE JSON_VALUE(payload, '$.tenant_id') = '<INJECTED_AT_RUNTIME>'\nGROUP BY window_start, window_end, trade_amount`;
  };

  return (
    <div className="bg-white border border-gray-200 rounded-lg p-6 shadow-sm max-w-4xl">
      <div className="flex items-center gap-3 mb-6 border-b pb-4">
        <div className="w-8 h-8 rounded-full bg-orange-100 flex items-center justify-center text-orange-600">
          ⚡
        </div>
        <div>
          <h2 className="text-lg font-bold text-gray-800">CEP Streaming Binding</h2>
          <p className="text-xs text-gray-500">Bind this Business Object to a continuous Redpanda/Flink stream.</p>
        </div>
      </div>

      <div className="grid grid-cols-2 gap-6">
        {/* Form Controls */}
        <div className="space-y-4">
          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase mb-1">Source Topic</label>
            <input 
              type="text" 
              value={config.topicName}
              onChange={e => setConfig({...config, topicName: e.target.value})}
              className="w-full border border-gray-300 rounded p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none font-mono"
            />
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase mb-1">Window Type</label>
              <select 
                value={config.windowType}
                onChange={e => setConfig({...config, windowType: e.target.value as any})}
                className="w-full border border-gray-300 rounded p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none cursor-pointer"
              >
                <option value="TUMBLE">Tumble (Fixed)</option>
                <option value="HOP">Hop (Sliding)</option>
                <option value="SESSION">Session</option>
              </select>
            </div>
            <div>
              <label className="block text-xs font-semibold text-gray-600 uppercase mb-1">Interval</label>
              <input 
                type="text" 
                value={config.windowInterval}
                onChange={e => setConfig({...config, windowInterval: e.target.value})}
                placeholder="e.g., 5 MINUTE"
                className="w-full border border-gray-300 rounded p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none font-mono"
              />
            </div>
          </div>

          <div>
            <label className="block text-xs font-semibold text-gray-600 uppercase mb-1">Watermark Lateness</label>
            <input 
              type="text" 
              value={config.watermarkDelay}
              onChange={e => setConfig({...config, watermarkDelay: e.target.value})}
              className="w-full border border-gray-300 rounded p-2 text-sm focus:ring-2 focus:ring-blue-500 outline-none font-mono"
            />
          </div>
          
          <button className="w-full bg-gray-900 text-white font-bold py-2 rounded shadow hover:bg-gray-800 transition-colors cursor-pointer">
            Save Streaming Binding
          </button>
        </div>

        {/* Compiler Preview */}
        <div className="bg-gray-900 rounded-lg p-4 flex flex-col h-full">
          <div className="flex justify-between items-center mb-2">
            <span className="text-xs font-semibold text-gray-400 uppercase tracking-wider">Generated Flink SQL Preview</span>
            <span className="flex h-3 w-3">
              <span className="animate-ping absolute inline-flex h-3 w-3 rounded-full bg-green-400 opacity-75"></span>
              <span className="relative inline-flex rounded-full h-3 w-3 bg-green-500"></span>
            </span>
          </div>
          <pre className="text-green-400 font-mono text-xs overflow-x-auto flex-1 mt-2">
            {generatePreviewSQL()}
          </pre>
          <div className="mt-4 pt-3 border-t border-gray-700 text-xs text-gray-500">
            * Tenant scoping and ABAC policies are automatically injected at compilation time.
          </div>
        </div>
      </div>
    </div>
  );
};

export default StreamingBindingPanel;
