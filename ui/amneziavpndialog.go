/* SPDX-License-Identifier: MIT
 *
 * Copyright (C) 2019-2022 WireGuard LLC. All Rights Reserved.
 */

package ui

import (
	"github.com/lxn/walk"

	"github.com/amnezia-vpn/amneziawg-windows-client/l18n"
	"github.com/amnezia-vpn/amneziawg-windows-client/manager"
	"github.com/amnezia-vpn/awg-windows/conf"
)

type AmneziaVPNDialog struct {
	*walk.Dialog
	textEdit   *walk.TextEdit
	text2Edit  *walk.TextEdit
	saveButton *walk.PushButton
	config     conf.Config
}

func runAmneziaVPNDialog(owner walk.Form, tunnel *manager.Tunnel) *conf.Config {
	dlg, err := newAmneziaVPNDialog(owner, tunnel)
	if showError(err, owner) {
		return nil
	}

	if dlg.Run() == walk.DlgCmdOK {
		return &dlg.config
	}

	return nil
}

func newAmneziaVPNDialog(owner walk.Form, tunnel *manager.Tunnel) (*AmneziaVPNDialog, error) {
	var err error
	var disposables walk.Disposables
	defer disposables.Treat()

	dlg := new(AmneziaVPNDialog)

	var title string
	if tunnel == nil {
		title = l18n.Sprintf("Create new tunnel")
	} else {
		title = l18n.Sprintf("Edit tunnel")
	}

	if tunnel == nil {
		// Creating a new tunnel, create a new private key and use the default template
		pk, _ := conf.NewPrivateKey()
		dlg.config = conf.Config{Interface: conf.Interface{PrivateKey: *pk}}
	} else {
		dlg.config, _ = tunnel.StoredConfig()
	}

	layout := walk.NewGridLayout()
	layout.SetSpacing(6)
	layout.SetMargins(walk.Margins{10, 10, 10, 10})
	layout.SetColumnStretchFactor(1, 3)

	if dlg.Dialog, err = walk.NewDialog(owner); err != nil {
		return nil, err
	}
	disposables.Add(dlg)
	dlg.SetIcon(owner.Icon())
	dlg.SetTitle(title)
	dlg.SetLayout(layout)
	dlg.SetMinMaxSize(walk.Size{500, 400}, walk.Size{0, 0})
	if icon, err := loadSystemIcon("imageres", -114, 32); err == nil {
		dlg.SetIcon(icon)
	}

	if dlg.textEdit, err = walk.NewTextEdit(dlg); err != nil {
		return nil, err
	}
	layout.SetRange(dlg.textEdit, walk.Rectangle{0, 0, 2, 1})

	if dlg.text2Edit, err = walk.NewTextEdit(dlg); err != nil {
		return nil, err
	}
	layout.SetRange(dlg.text2Edit, walk.Rectangle{0, 2, 2, 1})

	buttonsContainer, err := walk.NewComposite(dlg)
	if err != nil {
		return nil, err
	}
	layout.SetRange(buttonsContainer, walk.Rectangle{0, 3, 2, 1})
	buttonsContainer.SetLayout(walk.NewHBoxLayout())
	buttonsContainer.Layout().SetMargins(walk.Margins{})

	walk.NewHSpacer(buttonsContainer)

	if dlg.saveButton, err = walk.NewPushButton(buttonsContainer); err != nil {
		return nil, err
	}
	dlg.saveButton.SetText(l18n.Sprintf("&Save"))
	dlg.saveButton.Clicked().Attach(dlg.onSaveButtonClicked)

	cancelButton, err := walk.NewPushButton(buttonsContainer)
	if err != nil {
		return nil, err
	}
	cancelButton.SetText(l18n.Sprintf("Cancel"))
	cancelButton.Clicked().Attach(dlg.Cancel)

	dlg.SetCancelButton(cancelButton)
	dlg.SetDefaultButton(dlg.saveButton)

	if tunnel != nil {
		dlg.Starting().Attach(func() {
			dlg.textEdit.SetFocus()
		})
	}

	disposables.Spare()

	return dlg, nil
}

func (dlg *AmneziaVPNDialog) onSaveButtonClicked() {
	dlg.Accept()
}
