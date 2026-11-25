import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

const resources = {
  'pt-BR': {
    translation: {
      nexV2: {
        avatarLabel: 'Nex, assistente do Nexus',
        avatarMood: 'Estado {{mood}}',
        dragLabel: 'Mover janela do Nex',
        dragTitle: 'Arrastar Nex ou mover com setas',
        atlasAlt: 'Mascote Nex',
        labTitle: 'Laboratório Nex 2.0',
        moodLabel: 'Estado',
        behaviorLabel: 'Comportamento',
        reducedMotionLabel: 'Movimento reduzido',
      },
    },
  },
  en: {
    translation: {
      nexV2: {
        avatarLabel: 'Nex, Nexus assistant',
        avatarMood: 'State {{mood}}',
        dragLabel: 'Move Nex window',
        dragTitle: 'Drag Nex or move with arrow keys',
        atlasAlt: 'Nex mascot',
        labTitle: 'Nex 2.0 Lab',
        moodLabel: 'State',
        behaviorLabel: 'Behavior',
        reducedMotionLabel: 'Reduced motion',
      },
    },
  },
  es: {
    translation: {
      nexV2: {
        avatarLabel: 'Nex, asistente de Nexus',
        avatarMood: 'Estado {{mood}}',
        dragLabel: 'Mover ventana de Nex',
        dragTitle: 'Arrastrar Nex o mover con flechas',
        atlasAlt: 'Mascota Nex',
        labTitle: 'Laboratorio Nex 2.0',
        moodLabel: 'Estado',
        behaviorLabel: 'Comportamiento',
        reducedMotionLabel: 'Movimiento reducido',
      },
    },
  },
};

if (!i18n.isInitialized) {
  void i18n.use(initReactI18next).init({
    resources,
    lng: 'pt-BR',
    fallbackLng: 'en',
    interpolation: { escapeValue: false },
  });
}

export default i18n;
