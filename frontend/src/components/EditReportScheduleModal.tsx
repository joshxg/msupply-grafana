import React, { FC, useState, useEffect } from 'react';
import intl from 'react-intl-universal';
import { Modal, Button, ConfirmModal } from '@grafana/ui';

import { queryCache, useMutation } from 'react-query';
import { deleteSchedule, updateSchedule } from 'api';

import { css } from 'emotion';

import { PanelList } from './PanelList';
import { useToggle } from 'hooks';
import { Schedule } from 'common/types';
import { EditScheduleForm } from './Schedules/EditScheduleForm';
import { ScheduleKey } from 'common/enums';

type Props = {
  onClose: () => void;
  isOpen: boolean;
  reportSchedule: Schedule;
  datasourceID: number;
};

const modalAdjustments = css`
  top: 0;
  bottom: 0;
  width: 80%;
`;

const headerAdjustments = css`
  display: flex;
  flex: 1;
  justify-content: flex-end;
`;

export const EditReportScheduleModal: FC<Props> = ({ reportSchedule, onClose, isOpen, datasourceID }) => {
  const [schedule, setReportSchedule] = useState<Schedule>(reportSchedule);
  const [deleteAlertIsOpen, setDeleteAlertIsOpen] = useToggle(false);

  const [updateReportSchedule] = useMutation(updateSchedule, {
    onSuccess: () => queryCache.refetchQueries(['reportSchedules']),
  });

  const [deleteReportSchedule] = useMutation(deleteSchedule, {
    onSuccess: () => queryCache.refetchQueries(['reportSchedules']),
  });

  useEffect(() => {
    if (!schedule) {
      setReportSchedule(reportSchedule);
    }
  }, [schedule, reportSchedule]);

  // TODO: Handle error cases
  const onUpdateSchedule = (key: ScheduleKey, newValue: string | number) => {
    const newState: Schedule = { ...schedule, [key]: newValue };
    // Optimistically update state to reflect changes immediately in UI.
    setReportSchedule(newState);
    updateReportSchedule(newState);
  };

  const onConfirmDelete = () => {
    deleteReportSchedule(schedule);
    setDeleteAlertIsOpen();
    onClose();
  };

  return (
    <Modal
      className={modalAdjustments}
      onClickBackdrop={() => {}}
      title={intl.get('edit_report_schedule')}
      isOpen={isOpen}
      onDismiss={onClose}
    >
      <div className={headerAdjustments}>
        <EditScheduleForm schedule={schedule} onUpdate={onUpdateSchedule} />
        <Button size="md" variant="destructive" onClick={setDeleteAlertIsOpen}>
          {intl.get('delete')}
        </Button>
      </div>

      <PanelList schedule={reportSchedule} datasourceID={datasourceID} />

      <ConfirmModal
        isOpen={deleteAlertIsOpen}
        title={intl.get('delete_report_group')}
        body={intl.get('delete_report_group_question')}
        confirmText={intl.get('delete')}
        icon="exclamation-triangle"
        onConfirm={onConfirmDelete}
        onDismiss={setDeleteAlertIsOpen}
      />
    </Modal>
  );
};
