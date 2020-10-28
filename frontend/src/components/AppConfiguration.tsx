import React, { FC } from 'react';
import intl from 'react-intl-universal';
import { AppPluginMeta, PluginConfigPageProps } from '@grafana/data';
import { getBackendSrv } from '@grafana/runtime';
import { css } from 'emotion';
import { Button } from '@grafana/ui';
import { DisabledState } from './DisabledState';

import { FormValues } from '../types';
import { ConfigurationForm } from './ConfigurationForm';

interface Props extends PluginConfigPageProps<AppPluginMeta> {}

export const AppConfiguration: FC<Props> = (props: Props) => {
  const toggleAppState = (newState = true) => {
    getBackendSrv()
      .post(`/api/plugins/${props.plugin.meta.id}/settings`, {
        ...props.plugin.meta,
        enabled: newState,
        pinned: newState,
      })
      // Reload the current URL to update the app and show the sidebar
      // link and icon.
      .then(() => (window.location.href = window.location.href));
  };

  const onSubmit = (newJsonData: FormValues) => {
    getBackendSrv().post(`/api/plugins/msupply-datasource/resources/settings`, newJsonData);
    getBackendSrv().post(`/api/plugins/${props.plugin.meta.id}/settings`, {
      ...props.plugin.meta,
      jsonData: newJsonData,
    });
  };

  const defaultFormValues = {
    grafanaPassword: props.plugin.meta.jsonData?.grafanaPassword ?? '',
    grafanaUsername: props.plugin.meta.jsonData?.grafanaUsername ?? '',
    email: props.plugin.meta.jsonData?.email ?? '',
    emailPassword: props.plugin.meta.jsonData?.emailPassword ?? '',
  };

  const isEnabled = props.plugin.meta.enabled;

  return isEnabled ? (
    <div
      className={css`
        margin: auto;
      `}
    >
      <ConfigurationForm formValues={defaultFormValues} onSubmit={onSubmit} />
      <Button variant="destructive" onClick={() => toggleAppState(false)}>
        {intl.get('disable')}
      </Button>
    </div>
  ) : (
    <DisabledState toggle={toggleAppState} />
  );
};