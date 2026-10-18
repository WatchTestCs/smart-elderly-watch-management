<template>
  <div class="app-root">
    <care-config><router-view /></care-config>
  </div>
</template>

<script setup>
import { onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { monitorDatabase } from './auth'
const router = useRouter()
let stopMonitoring
onMounted(() => { stopMonitoring = monitorDatabase(router) })
onBeforeUnmount(() => stopMonitoring?.())
</script>

<style>

html,
body,
#app {
  min-height: 100%;
  margin: 0;
}

body {
  background: #f7f8f5;
  color: #263632;
  font-family: var(--care-font-family);
  font-optical-sizing: auto;
  font-variant-numeric: lining-nums tabular-nums;
  font-size: 16px;
  line-height: 1.65;
  -webkit-font-smoothing: antialiased;
}

* {
  box-sizing: border-box;
}

:root {
  --care-ink: #263632;
  --care-muted: #59665f;
  --care-primary: #235442;
  --care-primary-dark: #183e33;
  --care-canvas: #f7f8f5;
  --care-surface: #ffffff;
  --care-line: #e5ebe7;
  --care-shadow: none;
  --care-control-height: 52px;
  --care-control-line: #81917b;
  --care-motion-fast: 140ms;
  --care-motion-settle: 200ms;
  --care-ease: cubic-bezier(.2,.7,.3,1);
}

#app .el-button--primary.is-disabled { background: #e2e9dc; border-color: #c7d3c1; color: #59665f; }

button,
input,
textarea {
  font: inherit;
}

button { transition: background-color var(--care-motion-fast) var(--care-ease), color var(--care-motion-fast) var(--care-ease), border-color var(--care-motion-fast) var(--care-ease), transform var(--care-motion-fast) var(--care-ease); }
button:not(:disabled):active { transform: translateY(1px); }

:focus-visible {
  outline: 3px solid #235442;
  outline-offset: 2px;
}

#app .el-button--primary,
#app .el-button--primary:focus {
  background-color: var(--care-primary);
  border-color: var(--care-primary);
  color: #fff;
}

#app .el-button--primary:hover {
  background-color: var(--care-primary-dark);
  border-color: var(--care-primary-dark);
}

#app .el-button.is-link,
#app .el-button.is-link:focus {
  color: var(--care-primary);
}

#app .el-button.is-link:hover {
  color: var(--care-primary-dark);
}

#app .el-radio__input.is-checked .el-radio__inner {
  background-color: var(--care-primary);
  border-color: var(--care-primary);
}

#app .el-radio__input.is-checked + .el-radio__label {
  color: var(--care-primary);
}

#app .el-switch.is-checked .el-switch__core,
#app .el-pagination.is-background .el-pager li:not(.is-disabled).is-active {
  background-color: var(--care-primary);
  border-color: var(--care-primary);
}

/* The wrapper owns the field boundary; native inputs remain unframed. */
.el-input__inner { height: calc(var(--care-control-height) - 2px); font-size: 16px; }
#app .el-input__inner,
#app .el-select__input,
#app .el-range-input,
#app .el-input__inner:focus { border: 0; border-radius: 0; box-shadow: none; outline: none; background: transparent; }
#app .el-input__wrapper,
#app .el-select__wrapper,
#app .el-textarea__inner {
  min-height: var(--care-control-height);
  border: 1px solid var(--care-control-line);
  border-radius: 8px;
  box-shadow: none;
  transition: border-color var(--care-motion-fast) var(--care-ease), background-color var(--care-motion-fast) var(--care-ease);
}
#app .el-input__wrapper { padding: 0 14px; }
#app .el-select__wrapper { padding: 8px 14px; font-size: 16px; }
#app .el-textarea__inner { font-size: 16px; }
#app .el-input__wrapper:hover,
#app .el-select__wrapper:hover,
#app .el-textarea__inner:hover { border-color: var(--care-primary); }
#app .el-input__wrapper:focus-within,
#app .el-select__wrapper:focus-within,
#app .el-select__wrapper.is-focused,
#app .el-textarea__inner:focus {
  border-color: var(--care-primary);
  outline: 2px solid var(--care-primary);
  outline-offset: -1px;
}
#app .el-input.is-disabled .el-input__wrapper,
#app .el-select__wrapper.is-disabled { border-color: #cbd5c5; outline: none; background: #f1f3ef; }

/* A search group has one continuous frame, including its action segment. */
#app .el-input-group {
  display: inline-flex;
  min-height: var(--care-control-height);
  border: 1px solid var(--care-control-line);
  border-radius: 8px;
  background: #fff;
  box-shadow: none;
  overflow: hidden;
  transition: border-color var(--care-motion-fast) var(--care-ease);
}
#app .el-input-group:hover { border-color: var(--care-primary); }
#app .el-input-group:focus-within { border-color: var(--care-primary); outline: 2px solid var(--care-primary); outline-offset: -1px; }
#app .el-input-group .el-input__wrapper,
#app .el-input-group .el-input__wrapper:focus-within { min-width: 0; min-height: 0; border: 0; border-radius: 0; box-shadow: none; outline: none; }
#app .el-input-group__append { align-self: stretch; padding: 0; border: 0; border-radius: 0; box-shadow: none; background: transparent; }
#app .el-input-group .el-input-group__append .el-button {
  display: inline-flex;
  align-self: stretch;
  width: var(--care-control-height);
  min-width: var(--care-control-height);
  height: auto;
  min-height: 0;
  margin: 0;
  padding: 0;
  border: 0;
  border-radius: 0;
  box-shadow: none;
  outline: none;
  color: var(--care-primary);
  background: #f1f4ed;
}
#app .el-input-group .el-input-group__append .el-button:not(:disabled):hover { background: #e7eee3; }
#app .el-input-group .el-input-group__append .el-button:not(:disabled):active { transform: none; background: #dce7d6; }
#app .el-input-group.is-disabled { border-color: #cbd5c5; background: #f1f3ef; }
#app .el-input-group .el-input-group__append .el-button:disabled { background: #f1f3ef; color: #69796b; }

#app .el-input-number { height: var(--care-control-height); }
#app .el-input-number .el-input__wrapper { padding: 0 38px; }
#app .el-input-number__increase,
#app .el-input-number__decrease { border: 0; background: transparent; }
#app .el-input-number__increase:hover,
#app .el-input-number__decrease:hover { background: #edf3e8; }

.el-button { min-height: 44px; font-size: 15px; }
.el-button--small { min-height: 40px; font-size: 14px; }
.el-form-item__label { font-size: 15px; line-height: 1.5; }
.el-table { font-size: 16px; }
.el-table th.el-table__cell { font-size: 14px; }
.el-menu-item, .el-sub-menu__title { font-size: 16px; }
.el-tabs__item { font-size: 15px; }


.el-table {
  color: var(--care-ink);
  border-radius: 10px;
  overflow: hidden;
}

.el-table th.el-table__cell {
  background: #f6f8f6;
  color: #53645c;
  font-weight: 600;
}

.el-table td.el-table__cell,
.el-table th.el-table__cell {
  padding: 12px 0;
}

#app .care-app .el-table th { background: #f6f8f6; color: #53645c; font-weight: 600; }
#app .care-app .el-input__inner::placeholder { color: #59665f; opacity: 1; }
html.large-text #app .care-app .el-table,
html.large-text #app .care-app .el-input__inner,
html.large-text #app .care-app .el-form-item__label,
html.large-text #app .care-app .workspace-content p { font-size: 18px; }
html.large-text #app .care-app .el-menu-item,
html.large-text #app .care-app .el-sub-menu__title,
html.large-text #app .care-app .el-button { font-size: 17px; }

.el-table--border::after,
.el-table--group::after,
.el-table::before {
  background-color: var(--care-line);
}

.el-dialog {
  max-width: calc(100vw - 24px);
  border-radius: 14px;
  overflow: hidden;
  border: 1px solid #dce4da;
  box-shadow: 0 24px 80px rgba(16,43,34,.18), 0 4px 16px rgba(16,43,34,.08);
}

/* One short, interruptible response to navigation; no looping or scroll effects. */
.care-view-enter-active { transition: opacity var(--care-motion-settle) var(--care-ease), transform var(--care-motion-settle) var(--care-ease); }
.care-view-enter-from { opacity: 0; transform: translateY(5px); }
.dialog-fade-enter-active { animation: none !important; transition: opacity var(--care-motion-settle) var(--care-ease); }
.dialog-fade-leave-active { animation: none !important; transition: opacity var(--care-motion-fast) var(--care-ease); }
.dialog-fade-enter-from, .dialog-fade-leave-to { opacity: 0; }
.dialog-fade-enter-active .el-dialog { transition: transform var(--care-motion-settle) var(--care-ease); }
.dialog-fade-enter-from .el-dialog { transform: translateY(8px); }
.el-button.is-loading, .el-button.is-disabled { transform: none; }
.el-loading-mask { transition: opacity var(--care-motion-fast) var(--care-ease); }
.el-message { max-width: calc(100vw - 32px); min-width: 0; border-radius: 10px; }
.el-message__content { line-height: 1.7; overflow-wrap: anywhere; }
.el-message-box { width: 420px; max-width: calc(100vw - 32px); border-radius: 14px; }
.el-message-box__header { padding: 24px 56px 16px 24px; }
.el-message-box__content { padding: 8px 24px 20px; }
.el-message-box__btns { padding: 16px 24px; border-top: 1px solid var(--care-line); }

.el-dialog__header {
  border-bottom: 1px solid var(--care-line);
  padding: 20px 24px;
}

.el-dialog__body {
  padding: 24px;
}

.el-tag {
  border-radius: 999px;
}

.el-pagination {
  color: var(--care-muted);
}

.el-pagination button:hover,
.el-pagination__jump:hover {
  color: var(--care-primary);
}

.el-table__body tr:hover > td.el-table__cell {
  background: #f7faf8;
}

#app .roles {
  min-width: 0;
  padding: 22px;
  border: 1px solid var(--care-line);
  border-radius: 14px;
  background: var(--care-surface);
  box-shadow: var(--care-shadow);
}

#app .roles .sys-research {
  height: auto;
  margin-bottom: 18px;
  padding: 0;
}

#app .roles .row-search {
  display: flex;
  align-items: flex-end;
  flex-wrap: wrap;
  gap: 16px 20px;
  margin: 0;
}


#app .roles .el-table {
  max-width: 100%;
  border: 1px solid var(--care-line);
}

.care-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 32px 16px;
  color: var(--care-muted);
  text-align: center;
  font-size: 14px;
}

.care-empty-state strong { color: #53645c; font-size: 15px; font-weight: 600; }
.care-empty-state span { max-width: 520px; line-height: 1.65; }

#app .el-form-item__label {
  color: #53645c;
}

#app .el-form-item__error { color: #a4262c; font-size: 16px; line-height: 1.5; }
#app .el-form-item.is-error .el-input__wrapper, #app .el-form-item.is-error .el-select__wrapper, #app .el-form-item.is-error .el-input-group, #app .el-form-item.is-error .el-textarea__inner { border-color: #a4262c; outline-color: #a4262c; }
#app .el-form-item.is-error .el-input-group .el-input__wrapper { border: 0; outline: none; }
#app .el-form-item.is-required:not(.is-no-asterisk) > .el-form-item__label::before { color: #a4262c; }
.care-family .el-form-item__error, .auth-page .el-form-item__error, .register-page .el-form-item__error { position: static; padding-top: 6px; font-size: 16px; }
#app .el-radio__inner { width: 18px; height: 18px; border-color: #81917b; }
#app .el-radio__inner::after { width: 6px; height: 6px; }
.register-page .el-input__inner::placeholder { color: #59665f; opacity: 1; }

html.large-text { --care-control-height: 60px; }
html.large-text .care-family { --care-control-height: 64px; }

/* Family portal defaults are intentionally larger for older and low-vision users. */
.care-family {
  --care-control-height: 56px;
  font-size: 18px;
}

.care-family .el-button,
.care-family .el-button--small {
  min-height: 48px;
  padding-right: 18px;
  padding-left: 18px;
  font-size: 17px;
}

.care-family .el-button.is-link {
  min-height: 48px;
  padding: 10px 4px;
}

.care-family .el-input__inner,
.care-family .el-textarea__inner {
  min-height: 50px;
  font-size: 18px;
}

.care-family .el-form-item__label,
.care-family .el-table,
.care-family .el-table th.el-table__cell {
  font-size: 16px;
}

.care-family .el-table td.el-table__cell,
.care-family .el-table th.el-table__cell {
  padding-top: 16px;
  padding-bottom: 16px;
}

.care-family .el-menu-item {
  min-height: 56px;
  line-height: 56px;
  font-size: 18px;
}

.care-family .el-tabs__item {
  height: 54px;
  line-height: 54px;
  font-size: 17px;
}

.care-family .el-dialog__body {
  font-size: 17px;
  line-height: 1.7;
}

.care-family .record-heading { display: flex; justify-content: space-between; align-items: center; gap: 16px; margin-bottom: 20px; }
.care-family .record-heading h2 { margin: 0; color: #213c32; font-size: 22px; }
.care-family .el-alert { margin-bottom: 16px; }
.care-family .family-record-cards,
.care-family .family-record-empty { display: none; }
.care-family .status-normal { color: #236b3d; font-weight: 600; }
.care-family .status-danger { color: #a4262c; font-weight: 600; }
.care-family .el-tag--success { color: #236b3d; background: #edf7f0; border-color: #bad8c2; }
.care-family .el-tag--danger { color: #a4262c; background: #fff1f0; border-color: #e7b9b6; }
.care-family .care-empty-state,
.care-family .position-map-heading p,
.care-family .position-map-note,
.care-family .position-empty,
.care-family .unknown-state { color: #53645c; font-size: 16px; line-height: 1.7; }
.care-family .position-map { padding: 24px; }
.care-family .position-map-heading h2 { font-size: 21px; }
.care-family .el-dialog { display: flex; flex-direction: column; width: 92vw; max-width: 1000px; max-height: calc(100vh - 48px); margin: 24px auto !important; }
.care-family .el-dialog__header { flex-shrink: 0; padding-right: 76px; }
.care-family .el-dialog__body { min-height: 0; overflow-y: auto; }
.care-family .el-dialog__footer { flex-shrink: 0; padding: 16px 24px; border-top: 1px solid var(--care-line); }
.care-family .el-dialog__headerbtn { top: 8px; right: 12px; display: grid; place-items: center; width: 48px; height: 48px; font-size: 20px; }
.care-family .profile-dialog { max-width: 640px; }
.care-family .profile-dialog .el-input.is-disabled .el-input__inner { color: #53645c; -webkit-text-fill-color: #53645c; background: #f1f4eb; }
.profile-sex-options .el-select-dropdown__item { min-height: 48px; height: auto; line-height: 48px; color: #263632; font-size: 18px; }
html.large-text .profile-sex-options .el-select-dropdown__item { font-size: 20px; }

.auth-page { font-size: 18px; }
.auth-page .story-kicker { font-size: 15px; }
.auth-page .story-copy p { font-size: 18px; }
.auth-page .story-caption { color: #fff; font-size: 16px; text-shadow: 0 1px 4px rgba(0,0,0,.7); }
.auth-page .auth-eyebrow { color: #53645c; font-size: 16px; }
.auth-page .auth-intro { font-size: 17px; line-height: 1.65; }
.auth-page .login-form .el-input__inner { height: 52px; font-size: 18px; }
.care-family .el-input__inner::placeholder,
.care-family .el-textarea__inner::placeholder,
.auth-page .el-input__inner::placeholder,
.auth-page .el-textarea__inner::placeholder { color: #59665f; opacity: 1; }
.auth-page .auth-footnote,
.auth-page .auth-footer { color: #53645c; font-size: 15px; }
html.large-text .auth-page .story-kicker,
html.large-text .auth-page .auth-eyebrow,
html.large-text .auth-page .auth-footnote,
html.large-text .auth-page .auth-footer { font-size: 18px; }
html.large-text .auth-page .story-copy h1 { font-size: 56px; }
html.large-text .auth-page .story-copy p,
html.large-text .auth-page .auth-intro { font-size: 20px; }
html.large-text .auth-page .login-form .el-input__inner { height: 58px; font-size: 20px; }
html.large-text .care-family .care-empty-state,
html.large-text .care-family .position-map-heading p,
html.large-text .care-family .position-map-note,
html.large-text .care-family .position-empty,
html.large-text .care-family .unknown-state { font-size: 18px; }

@media (max-width: 1100px) {
  #app .care-family .roles { padding: 0; border: 0; background: transparent; box-shadow: none; }
  .care-family .family-record-cards { display: grid; gap: 14px; }
  .care-family .family-record-empty { display: block; padding: 24px 16px; border: 1px solid var(--care-line); border-radius: 12px; background: #fff; color: #33453c; font-size: 17px; line-height: 1.7; text-align: center; }
  .care-family .desktop-records { display: none; }
  .care-family .family-record-card { padding: 18px; border: 1px solid var(--care-line); border-radius: 12px; background: #fff; }
  .care-family .family-record-card h2 { margin: 0 0 12px; color: var(--care-ink); font-size: 19px; }
  .care-family .family-record-card dl { display: grid; grid-template-columns: minmax(100px, auto) 1fr; gap: 10px 12px; margin: 0 0 14px; }
  .care-family .family-record-card dt { color: #59665f; font-size: 15px; }
  .care-family .family-record-card dd { margin: 0; color: #263632; font-size: 17px; }
  .care-family .family-record-card dd { min-width: 0; overflow-wrap: anywhere; }
  .care-family .family-record-card dd.status-danger { color: #a4262c; }
  .care-family .family-record-card dd.status-normal { color: #236b3d; }
  .care-family .family-record-card .el-button { width: 100%; }
  .care-family .position-map { padding: 18px 14px; }
}

html.large-text .care-family .family-record-card h2 { font-size: 21px; }
html.large-text .care-family .family-record-card dt { font-size: 17px; }
html.large-text .care-family .family-record-card dd { font-size: 19px; }

html.large-text .care-family .el-table,
html.large-text .care-family .el-form-item__label { font-size: 18px; }
html.large-text .care-family .el-button,
html.large-text .care-family .el-input__inner,
html.large-text .care-family .el-textarea__inner { font-size: 19px; }
html.large-text .care-family .el-menu-item { font-size: 20px; }
html.large-text .care-family .care-sidebar .sidebar-label { font-size: 17px; }
html.large-text .care-family .care-sidebar .sidebar-note { font-size: 16px; }
html.large-text .care-family .care-sidebar .care-menu .el-menu-item {
  padding-right: 18px;
  padding-left: 18px;
}

html.large-text .care-header { min-height: 80px; height: auto; }
@media (min-width: 901px) {
  html.large-text .care-family .workspace-layout { grid-template-columns: 280px minmax(0, 1fr); }
}

@media (prefers-reduced-motion: reduce) {
  *, *::before, *::after {
    scroll-behavior: auto !important;
    animation-duration: .01ms !important;
    animation-iteration-count: 1 !important;
    transition-duration: .01ms !important;
  }
}

@media (max-width: 600px) {
  #app .roles { padding: 16px 12px; border-radius: 12px; }
  #app .roles .sys-research { margin-bottom: 14px; }
    .el-dialog__header { padding: 16px 18px; }
  .el-dialog__body { padding: 18px; }
  .el-dialog__footer { padding: 10px 18px 18px; }
  .el-pagination { white-space: normal; }
  #app .el-pagination__sizes, #app .el-pagination__jump { display: none; }
  html.large-text .care-header { min-height: 72px; }
  html.large-text .care-family .workspace-main { padding-right: 12px; padding-left: 12px; }
}
</style>

<style>
:root { --el-color-primary:#235442; --el-color-success:#235442; --el-font-family:var(--care-font-family); --el-border-radius-base:8px; }
</style>
