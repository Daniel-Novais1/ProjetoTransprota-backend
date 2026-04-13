import React, { useEffect } from 'react';
import { toast, ToastContainer } from 'react-toastify';
import { useRealtime } from '@/hooks/useRealtime';
import 'react-toastify/dist/ReactToastify.css';

export default function AlertToasts() {
  const { data, clearBIAlert, clearGeofenceAlert } = useRealtime();

  useEffect(() => {
    // Alerta de BI - Engarrafamento Detectado
    if (data.biAlert) {
      toast.error(
        `⚠️ ENGARRAFAMENTO DETECTADO!\n\nVelocidade média: ${data.biAlert.avg_speed.toFixed(1)} km/h\nBaseline: ${data.biAlert.baseline.toFixed(1)} km/h\nRedução: ${data.biAlert.reduction.toFixed(1)}%\nÔnibus monitorados: ${data.biAlert.bus_count}`,
        {
          position: 'top-right',
          autoClose: 10000,
          hideProgressBar: false,
          closeOnClick: true,
          pauseOnHover: true,
          draggable: true,
          onClose: clearBIAlert,
          style: {
            background: '#fef2f2',
            borderLeft: '4px solid #ef4444',
          },
        }
      );
    }

    // Alerta de Geofencing - Saída de Rota
    if (data.geofenceAlert) {
      toast.warning(
        `🚨 ALERTA DE GEOFENCING!\n\nÔnibus: ${data.geofenceAlert.device_id}\nCerca: ${data.geofenceAlert.fence}\nLat: ${data.geofenceAlert.lat.toFixed(6)}\nLng: ${data.geofenceAlert.lng.toFixed(6)}`,
        {
          position: 'top-left',
          autoClose: 15000,
          hideProgressBar: false,
          closeOnClick: true,
          pauseOnHover: true,
          draggable: true,
          onClose: clearGeofenceAlert,
          style: {
            background: '#fff7ed',
            borderLeft: '4px solid #f97316',
          },
        }
      );
    }
  }, [data.biAlert, data.geofenceAlert, clearBIAlert, clearGeofenceAlert]);

  return (
    <ToastContainer
      position="top-right"
      autoClose={5000}
      hideProgressBar={false}
      newestOnTop={false}
      closeOnClick
      rtl={false}
      pauseOnFocusLoss
      draggable
      pauseOnHover
      theme="light"
    />
  );
}
