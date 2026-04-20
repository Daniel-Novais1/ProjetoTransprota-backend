import { renderHook, act } from '@testing-library/react';
import { useBusTracker, fetchNearbyBuses, fetchAllBuses, fetchDensityReport } from './useBusTracker';

// ============================================================================
// REACT HOOK UNIT TESTS - useBusTracker
// ============================================================================

// Mock do navigator.geolocation
const mockGeolocation = {
	watchPosition: jest.fn(),
	clearWatch: jest.fn(),
	getCurrentPosition: jest.fn(),
};

Object.defineProperty(global.navigator, 'geolocation', {
	writable: true,
	value: mockGeolocation,
});

// Mock do fetch global
global.fetch = jest.fn();

describe('useBusTracker Hook', () => {
	beforeEach(() => {
		jest.clearAllMocks();
		jest.useFakeTimers();
	});

	afterEach(() => {
		jest.useRealTimers();
	});

	test('deve iniciar rastreamento geográfico ao montar', () => {
		const { result } = renderHook(() =>
			useBusTracker('user-123', 'device-hash-abc123', '001', {
				enabled: true,
				intervalMs: 5000,
				apiUrl: '/api/v1/bus-update',
			})
		);

		expect(mockGeolocation.watchPosition).toHaveBeenCalledWith(
			expect.any(Function),
			expect.any(Function),
			{
				enableHighAccuracy: true,
				timeout: 10000,
				maximumAge: 5000,
			}
		);

		expect(result.current.isTracking).toBe(true);
	});

	test('deve parar rastreamento ao desmontar', () => {
		const watchId = 12345;
		mockGeolocation.watchPosition.mockReturnValue(watchId);

		const { unmount } = renderHook(() =>
			useBusTracker('user-123', 'device-hash-abc123', '001')
		);

		unmount();

		expect(mockGeolocation.clearWatch).toHaveBeenCalledWith(watchId);
	});

	test('não deve iniciar rastreamento se disabled', () => {
		renderHook(() =>
			useBusTracker('user-123', 'device-hash-abc123', '001', {
				enabled: false,
			})
		);

		expect(mockGeolocation.watchPosition).not.toHaveBeenCalled();
	});

	test('deve enviar posição para API ao receber geolocalização', (done) => {
		const mockPosition = {
			coords: {
				latitude: -16.6869,
				longitude: -49.2648,
				speed: 10.5,
				heading: 45,
			},
			timestamp: Date.now(),
		};

		(global.fetch as jest.Mock).mockResolvedValue({
			ok: true,
			json: async () => ({ msg: 'Bus update received' }),
		});

		const { result } = renderHook(() =>
			useBusTracker('user-123', 'device-hash-abc123', '001')
		);

		// Simular callback de sucesso do watchPosition
		const successCallback = mockGeolocation.watchPosition.mock.calls[0][0];
		act(() => {
			successCallback(mockPosition);
		});

		setTimeout(() => {
			expect(global.fetch).toHaveBeenCalledWith('/api/v1/bus-update', {
				method: 'POST',
				headers: {
					'Content-Type': 'application/json',
				},
				body: JSON.stringify({
					user_id: 'user-123',
					device_hash: 'device-hash-abc123',
					lat: mockPosition.coords.latitude,
					lng: mockPosition.coords.longitude,
					route_id: '001',
					speed: mockPosition.coords.speed,
					heading: mockPosition.coords.heading,
					is_on_bus: true,
					occupancy: 'unknown',
					terminal_id: 't01',
				}),
			});
			done();
		}, 100);
	});

	test('deve chamar onError ao falhar geolocalização', () => {
		const onError = jest.fn();
		const mockError = new Error('Permission denied');

		mockGeolocation.watchPosition.mockImplementation((_, errorCb) => {
			errorCb(mockError);
		});

		renderHook(() =>
			useBusTracker('user-123', 'device-hash-abc123', '001', {
				onError,
			})
		);

		expect(onError).toHaveBeenCalledWith(mockError);
	});

	test('startTracking deve iniciar rastreamento manualmente', () => {
		const { result } = renderHook(() =>
			useBusTracker('user-123', 'device-hash-abc123', '001', {
				enabled: false,
			})
		);

		act(() => {
			result.current.startTracking();
		});

		expect(mockGeolocation.watchPosition).toHaveBeenCalled();
	});

	test('stopTracking deve parar rastreamento manualmente', () => {
		const watchId = 12345;
		mockGeolocation.watchPosition.mockReturnValue(watchId);

		const { result } = renderHook(() =>
			useBusTracker('user-123', 'device-hash-abc123', '001')
		);

		act(() => {
			result.current.stopTracking();
		});

		expect(mockGeolocation.clearWatch).toHaveBeenCalledWith(watchId);
	});
});

describe('Funções auxiliares do useBusTracker', () => {
	beforeEach(() => {
		jest.clearAllMocks();
	});

	describe('fetchNearbyBuses', () => {
		test('deve buscar ônibus próximos com parâmetros corretos', async () => {
			(global.fetch as jest.Mock).mockResolvedValue({
				ok: true,
				json: async () => ({ cnt: 5, buses: [] }),
			});

			await fetchNearbyBuses(-16.6869, -49.2648, 5000, '001');

			expect(global.fetch).toHaveBeenCalledWith(
				'/api/v1/bus-locations?lat=-16.6869&lng=-49.2648&radius=5000&route=001'
			);
		});

		test('deve buscar ônibus próximos sem filtro de rota', async () => {
			(global.fetch as jest.Mock).mockResolvedValue({
				ok: true,
				json: async () => ({ cnt: 10, buses: [] }),
			});

			await fetchNearbyBuses(-16.6869, -49.2648, 5000);

			expect(global.fetch).toHaveBeenCalledWith(
				'/api/v1/bus-locations?lat=-16.6869&lng=-49.2648&radius=5000'
			);
		});

		test('deve lançar erro se API falhar', async () => {
			(global.fetch as jest.Mock).mockResolvedValue({
				ok: false,
				statusText: 'Internal Server Error',
			});

			await expect(
				fetchNearbyBuses(-16.6869, -49.2648, 5000)
			).rejects.toThrow('HTTP 500: Internal Server Error');
		});
	});

	describe('fetchAllBuses', () => {
		test('deve buscar todos os ônibus ativos', async () => {
			(global.fetch as jest.Mock).mockResolvedValue({
				ok: true,
				json: async () => ({ cnt: 20, buses: [] }),
			});

			await fetchAllBuses();

			expect(global.fetch).toHaveBeenCalledWith('/api/v1/buses/all');
		});

		test('deve lançar erro se API falhar', async () => {
			(global.fetch as jest.Mock).mockResolvedValue({
				ok: false,
				statusText: 'Service Unavailable',
			});

			await expect(fetchAllBuses()).rejects.toThrow('HTTP 503: Service Unavailable');
		});
	});

	describe('fetchDensityReport', () => {
		test('deve buscar relatório de densidade com parâmetro padrão', async () => {
			(global.fetch as jest.Mock).mockResolvedValue({
				ok: true,
				json: async () => ({ densities: [], hours: 24 }),
			});

			await fetchDensityReport();

			expect(global.fetch).toHaveBeenCalledWith('/api/v1/analytics/density?hours=24');
		});

		test('deve buscar relatório de densidade com parâmetro customizado', async () => {
			(global.fetch as jest.Mock).mockResolvedValue({
				ok: true,
				json: async () => ({ densities: [], hours: 48 }),
			});

			await fetchDensityReport(48);

			expect(global.fetch).toHaveBeenCalledWith('/api/v1/analytics/density?hours=48');
		});

		test('deve lançar erro se API falhar', async () => {
			(global.fetch as jest.Mock).mockResolvedValue({
				ok: false,
				statusText: 'Bad Gateway',
			});

			await expect(fetchDensityReport()).rejects.toThrow('HTTP 502: Bad Gateway');
		});
	});
});
