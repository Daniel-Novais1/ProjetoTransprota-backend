import React, { useState, useRef, useEffect } from 'react';
import { DivIcon } from 'leaflet';
import axios from 'axios';

interface UserReport {
  tipo_problema: string;
  descricao: string;
  latitude: number;
  longitude: number;
  bus_line?: string;
}

interface ReportButtonProps {
  onReportSubmit?: (report: UserReport) => void;
  currentLocation?: { lat: number; lng: number };
  currentBusLine?: string;
  isDarkMode?: boolean;
}

const ReportButton: React.FC<ReportButtonProps> = ({
  onReportSubmit,
  currentLocation,
  currentBusLine,
  isDarkMode = false
}) => {
  const [isOpen, setIsOpen] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [selectedProblem, setSelectedProblem] = useState('');
  const [descricao, setDescricao] = useState('');
  const [showSuccess, setShowSuccess] = useState(false);
  const [error, setError] = useState('');
  const buttonRef = useRef<HTMLDivElement>(null);

  // Detectar clique fora para fechar menu
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (buttonRef.current && !buttonRef.current.contains(event.target as Node)) {
        setIsOpen(false);
      }
    };

    if (isOpen) {
      document.addEventListener('mousedown', handleClickOutside);
    }

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [isOpen]);

  // Haptic feedback para mobile
  const triggerHaptic = () => {
    if ('vibrate' in navigator) {
      navigator.vibrate(50); // Vibração curta de 50ms
    }
  };

  // Visual feedback toast
  const showSuccessToast = () => {
    setShowSuccess(true);
    triggerHaptic();
    setTimeout(() => setShowSuccess(false), 3000);
  };

  // Enviar denúncia
  const handleSubmit = async () => {
    if (!selectedProblem) {
      setError('Selecione um tipo de problema');
      return;
    }

    if (!currentLocation) {
      setError('Localização não disponível');
      return;
    }

    setIsSubmitting(true);
    setError('');

    try {
      const report: UserReport = {
        tipo_problema: selectedProblem,
        descricao: descricao.trim(),
        latitude: currentLocation.lat,
        longitude: currentLocation.lng,
        bus_line: currentBusLine
      };

      const response = await axios.post('http://localhost:8080/api/v1/reports', report);
      
      if (response.status === 201) {
        showSuccessToast();
        setSelectedProblem('');
        setDescricao('');
        setIsOpen(false);
        
        // Callback para componente pai
        if (onReportSubmit) {
          onReportSubmit(report);
        }
      }
    } catch (err: any) {
      if (err.response?.status === 429) {
        setError('Aguarde 5 minutos antes de enviar outra denúncia');
      } else {
        setError('Erro ao enviar denúncia. Tente novamente.');
      }
    } finally {
      setIsSubmitting(false);
    }
  };

  // Estilos dinâmicos baseados no tema
  const fabStyle = {
    position: 'fixed' as const,
    bottom: '20px',
    right: '20px',
    width: '56px',
    height: '56px',
    borderRadius: '50%',
    backgroundColor: isDarkMode ? '#ef4444' : '#dc2626',
    color: 'white',
    border: 'none',
    cursor: 'pointer',
    boxShadow: '0 4px 12px rgba(220, 38, 38, 0.3)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    fontSize: '24px',
    transition: 'all 0.3s ease',
    zIndex: 1000,
    transform: isOpen ? 'rotate(45deg)' : 'rotate(0deg)'
  };

  const menuStyle = {
    position: 'fixed' as const,
    bottom: '90px',
    right: '20px',
    backgroundColor: isDarkMode ? 'rgba(17, 24, 39, 0.95)' : 'rgba(255, 255, 255, 0.95)',
    backdropFilter: 'blur(10px)',
    borderRadius: '12px',
    padding: '16px',
    boxShadow: '0 4px 20px rgba(0,0,0,0.15)',
    border: `1px solid ${isDarkMode ? '#374151' : '#e5e7eb'}`,
    minWidth: '280px',
    zIndex: 999,
    opacity: isOpen ? 1 : 0,
    transform: isOpen ? 'translateY(0)' : 'translateY(20px)',
    pointerEvents: isOpen ? 'auto' : 'none' as const,
    transition: 'all 0.3s ease'
  };

  const buttonStyle = {
    backgroundColor: isDarkMode ? '#374151' : '#f3f4f6',
    color: isDarkMode ? '#f3f4f6' : '#111827',
    border: `1px solid ${isDarkMode ? '#4b5563' : '#d1d5db'}`,
    borderRadius: '8px',
    padding: '12px 16px',
    margin: '4px 0',
    cursor: 'pointer',
    width: '100%',
    textAlign: 'left' as const,
    fontSize: '14px',
    transition: 'all 0.2s ease'
  };

  const activeButtonStyle = {
    ...buttonStyle,
    backgroundColor: isDarkMode ? '#dc2626' : '#ef4444',
    color: 'white',
    borderColor: isDarkMode ? '#b91c1c' : '#dc2626'
  };

  const textareaStyle = {
    backgroundColor: isDarkMode ? '#1f2937' : '#ffffff',
    color: isDarkMode ? '#f3f4f6' : '#111827',
    border: `1px solid ${isDarkMode ? '#374151' : '#d1d5db'}`,
    borderRadius: '8px',
    padding: '12px',
    fontSize: '14px',
    width: '100%',
    minHeight: '80px',
    resize: 'vertical' as const,
    fontFamily: 'inherit'
  };

  const submitButtonStyle = {
    backgroundColor: isSubmitting ? '#6b7280' : '#dc2626',
    color: 'white',
    border: 'none',
    borderRadius: '8px',
    padding: '12px 16px',
    fontSize: '14px',
    fontWeight: '600',
    cursor: isSubmitting ? 'not-allowed' : 'pointer',
    width: '100%',
    marginTop: '8px',
    transition: 'all 0.2s ease'
  };

  const successToastStyle = {
    position: 'fixed' as const,
    top: '20px',
    right: '20px',
    backgroundColor: '#10b981',
    color: 'white',
    padding: '12px 16px',
    borderRadius: '8px',
    fontSize: '14px',
    fontWeight: '600',
    boxShadow: '0 4px 12px rgba(16, 185, 129, 0.3)',
    zIndex: 2000,
    opacity: showSuccess ? 1 : 0,
    transform: showSuccess ? 'translateX(0)' : 'translateX(100%)',
    transition: 'all 0.3s ease'
  };

  const problemTypes = [
    { id: 'Lotado', label: 'Ônibus Lotado', icon: ' crowd', color: '#f59e0b' },
    { id: 'Atrasado', label: 'Ônibus Atrasado', icon: ' clock', color: '#3b82f6' },
    { id: 'Perigo', label: 'Situação de Perigo', icon: ' warning', color: '#ef4444' }
  ];

  return (
    <div ref={buttonRef}>
      {/* FAB Button */}
      <button
        style={fabStyle}
        onClick={() => {
          setIsOpen(!isOpen);
          triggerHaptic();
        }}
        title="Reportar Problema"
      >
        {isOpen ? '×' : '+'}
      </button>

      {/* Menu de Report */}
      {isOpen && (
        <div style={menuStyle}>
          <h3 style={{ 
            margin: '0 0 12px 0', 
            fontSize: '16px', 
            fontWeight: '700',
            color: isDarkMode ? '#f3f4f6' : '#111827'
          }}>
            Reportar Problema
          </h3>

          {currentBusLine && (
            <div style={{ 
              fontSize: '12px', 
              color: isDarkMode ? '#9ca3af' : '#6b7280',
              marginBottom: '8px',
              padding: '4px 8px',
              backgroundColor: isDarkMode ? '#374151' : '#f3f4f6',
              borderRadius: '4px'
            }}>
              Linha: {currentBusLine}
            </div>
          )}

          {/* Tipos de Problema */}
          <div style={{ marginBottom: '12px' }}>
            <label style={{ 
              fontSize: '12px', 
              fontWeight: '600',
              color: isDarkMode ? '#9ca3af' : '#6b7280',
              display: 'block',
              marginBottom: '4px'
            }}>
              Tipo de Problema:
            </label>
            {problemTypes.map(type => (
              <button
                key={type.id}
                style={selectedProblem === type.id ? activeButtonStyle : buttonStyle}
                onClick={() => {
                  setSelectedProblem(type.id);
                  triggerHaptic();
                }}
              >
                <span style={{ marginRight: '8px' }}>{type.icon}</span>
                {type.label}
              </button>
            ))}
          </div>

          {/* Descrição */}
          <div style={{ marginBottom: '12px' }}>
            <label style={{ 
              fontSize: '12px', 
              fontWeight: '600',
              color: isDarkMode ? '#9ca3af' : '#6b7280',
              display: 'block',
              marginBottom: '4px'
            }}>
              Descrição (opcional):
            </label>
            <textarea
              style={textareaStyle}
              placeholder="Detalhes sobre o problema..."
              value={descricao}
              onChange={(e) => setDescricao(e.target.value)}
              maxLength={200}
            />
            <div style={{ 
              fontSize: '10px', 
              color: isDarkMode ? '#6b7280' : '#9ca3af',
              textAlign: 'right',
              marginTop: '2px'
            }}>
              {descricao.length}/200
            </div>
          </div>

          {/* Botão Submit */}
          <button
            style={submitButtonStyle}
            onClick={handleSubmit}
            disabled={isSubmitting || !selectedProblem}
          >
            {isSubmitting ? 'Enviando...' : 'Enviar Denúncia'}
          </button>

          {/* Erro */}
          {error && (
            <div style={{ 
              color: '#ef4444', 
              fontSize: '12px',
              marginTop: '8px',
              textAlign: 'center'
            }}>
              {error}
            </div>
          )}
        </div>
      )}

      {/* Success Toast */}
      <div style={successToastStyle}>
        Denúncia registrada com sucesso!
      </div>
    </div>
  );
};

export default ReportButton;
