#!/usr/bin/env node

/**
 * Скрипт для экспорта дизайн-токенов из Figma
 * Использование: node scripts/figma-tokens.js
 */

const fs = require('fs');
const path = require('path');

// Конфигурация Figma API
const FIGMA_CONFIG = {
  fileId: 'YOUR_FIGMA_FILE_ID', // Замените на ID вашего файла
  token: 'YOUR_FIGMA_TOKEN',    // Ваш токен доступа
  outputDir: './src/styles/tokens',
};

/**
 * Получение данных из Figma API
 */
async function fetchFigmaData() {
  const url = `https://api.figma.com/v1/files/${FIGMA_CONFIG.fileId}`;
  
  try {
    const response = await fetch(url, {
      headers: {
        'X-Figma-Token': FIGMA_CONFIG.token,
      },
    });
    
    if (!response.ok) {
      throw new Error(`HTTP error! status: ${response.status}`);
    }
    
    return await response.json();
  } catch (error) {
    console.error('Ошибка при получении данных из Figma:', error);
    throw error;
  }
}

/**
 * Извлечение цветов из узлов Figma
 */
function extractColors(nodes) {
  const colors = new Map();
  
  function traverseNode(node) {
    if (node.fills) {
      node.fills.forEach(fill => {
        if (fill.type === 'SOLID' && fill.color) {
          const colorName = node.name.toLowerCase().replace(/\s+/g, '-');
          const hexColor = rgbToHex(fill.color);
          colors.set(colorName, hexColor);
        }
      });
    }
    
    if (node.children) {
      node.children.forEach(traverseNode);
    }
  }
  
  Object.values(nodes).forEach(traverseNode);
  return colors;
}

/**
 * Конвертация RGB в HEX
 */
function rgbToHex(rgb) {
  const r = Math.round(rgb.r * 255);
  const g = Math.round(rgb.g * 255);
  const b = Math.round(rgb.b * 255);
  return `#${r.toString(16).padStart(2, '0')}${g.toString(16).padStart(2, '0')}${b.toString(16).padStart(2, '0')}`;
}

/**
 * Генерация CSS файла с токенами
 */
function generateCSSFile(colors) {
  let css = ':root {\n';
  
  colors.forEach((value, key) => {
    css += `  --color-${key}: ${value};\n`;
  });
  
  css += '}\n';
  return css;
}

/**
 * Основная функция
 */
async function main() {
  try {
    console.log('🎨 Экспорт дизайн-токенов из Figma...');
    
    // Создаем директорию для токенов
    const outputDir = path.resolve(FIGMA_CONFIG.outputDir);
    if (!fs.existsSync(outputDir)) {
      fs.mkdirSync(outputDir, { recursive: true });
    }
    
    // Получаем данные из Figma
    const figmaData = await fetchFigmaData();
    
    // Извлекаем цвета
    const colors = extractColors(figmaData.document.children);
    
    // Генерируем CSS
    const cssContent = generateCSSFile(colors);
    
    // Сохраняем файл
    const outputFile = path.join(outputDir, 'figma-tokens.css');
    fs.writeFileSync(outputFile, cssContent);
    
    console.log(`✅ Токены экспортированы в ${outputFile}`);
    console.log(`📊 Найдено ${colors.size} цветов`);
    
  } catch (error) {
    console.error('❌ Ошибка:', error.message);
    process.exit(1);
  }
}

// Запуск скрипта
if (require.main === module) {
  main();
}

module.exports = {
  fetchFigmaData,
  extractColors,
  generateCSSFile,
};


