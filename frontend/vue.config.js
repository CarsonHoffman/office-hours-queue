module.exports = {
	pages: {
		index: {
			entry: 'src/main.ts',
			title: 'EECS Office Hours',
		},
	},
	devServer: {
		proxy: {
			'^/api': {
				target: 'https://lvh.me:8080',
				ws: true,
				// changeOrigin didn't work for WebSocket connections.
				onProxyReqWs: function(request) {
					request.setHeader('origin', 'https://lvh.me:8080');
				},
			},
		},
	},
};
