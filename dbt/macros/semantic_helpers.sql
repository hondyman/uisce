{% macro semantic(term_id) -%}
  {{ return('semantic("' ~ term_id ~ '")') }}
{%- endmacro %}
