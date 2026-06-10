#!/usr/bin/env bash
# Prueba de Nexo interactivo
echo "Iniciando Nexo por 8 segundos..."
echo -e "Hola Nexo\n¿Qué puedes hacer?\nsalir" | timeout 8 /usr/local/bin/nexo 2>/dev/null
echo ""
echo "--- FIN PRUEBA ---"