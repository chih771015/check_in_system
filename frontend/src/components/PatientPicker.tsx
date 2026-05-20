import { useEffect, useMemo, useRef, useState } from 'react';
import { Select, Spin } from 'antd';
import { useTranslation } from 'react-i18next';
import { getPatients } from '../api/patients';
import type { Patient } from '../types';

interface PatientPickerProps {
  /** Selected patient id (undefined when nothing chosen yet). */
  value?: number;
  onChange: (patientId: number) => void;
  /** Optional disabled state, e.g. while parent form is submitting. */
  disabled?: boolean;
  /** Override placeholder. Falls back to i18n. */
  placeholder?: string;
  /** Width applied to the underlying Select. */
  style?: React.CSSProperties;
}

const PAGE_SIZE = 20;
const DEBOUNCE_MS = 250;

/**
 * PatientPicker is a typeahead Select that loads patients from the backend on
 * mount and re-queries (debounced) as the user types. Used inside the schedule
 * editor to attach patients to a schedule slot.
 */
export default function PatientPicker({ value, onChange, disabled, placeholder, style }: PatientPickerProps) {
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [options, setOptions] = useState<Patient[]>([]);
  const [search, setSearch] = useState('');
  const debounceRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Always re-query when search changes (incl. empty for initial load).
  useEffect(() => {
    if (debounceRef.current) clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      setLoading(true);
      getPatients({ search, page: 1, pageSize: PAGE_SIZE })
        .then((res) => setOptions(res.data))
        .catch(() => setOptions([]))
        .finally(() => setLoading(false));
    }, search ? DEBOUNCE_MS : 0);
    return () => {
      if (debounceRef.current) clearTimeout(debounceRef.current);
    };
  }, [search]);

  const selectOptions = useMemo(
    () =>
      options.map((p) => ({
        value: p.id,
        label: `${p.name} (${p.idType.toUpperCase()}:${p.idNumber})`,
      })),
    [options],
  );

  return (
    <Select
      showSearch
      filterOption={false} // server-side filter
      value={value}
      onChange={onChange}
      onSearch={setSearch}
      placeholder={placeholder ?? t('patients.searchPlaceholder')}
      notFoundContent={loading ? <Spin size="small" /> : null}
      options={selectOptions}
      disabled={disabled}
      style={style ?? { width: '100%' }}
    />
  );
}
