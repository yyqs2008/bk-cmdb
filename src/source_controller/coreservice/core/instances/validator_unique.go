/*
 * Tencent is pleased to support the open source community by making 蓝鲸 available.,
 * Copyright (C) 2017-2018 THL A29 Limited, a Tencent company. All rights reserved.
 * Licensed under the MIT License (the ",License",); you may not use this file except
 * in compliance with the License. You may obtain a copy of the License at
 * http://opensource.org/licenses/MIT
 * Unless required by applicable law or agreed to in writing, software distributed under
 * the License is distributed on an ",AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND,
 * either express or implied. See the License for the specific language governing permissions and
 * limitations under the License.
 */

package instances

import (
	"configcenter/src/common"
	"configcenter/src/common/blog"
	"configcenter/src/common/mapstr"
	"configcenter/src/common/metadata"
	"configcenter/src/common/universalsql/mongo"
	"configcenter/src/source_controller/coreservice/core"
)

// validCreateUnique  valid create inst data unique
func (valid *validator) validCreateUnique(ctx core.ContextParams, instanceData mapstr.MapStr, instMedataData metadata.Metadata, instanceManager *instanceManager) error {
	uniqueAttr, err := valid.dependent.SearchUnique(ctx, valid.objID)
	if nil != err {
		blog.Errorf("[validCreateUnique] search [%s] unique error %v", valid.objID, err)
		return err
	}

	if 0 >= len(uniqueAttr) {
		blog.Warnf("[validCreateUnique] there're not unique constraint for %s, return", valid.objID)
		return nil
	}

	for _, unique := range uniqueAttr {
		// retrive unique value
		uniquekeys := map[string]bool{}
		for _, key := range unique.Keys {
			switch key.Kind {
			case metadata.UniqueKeyKindProperty:
				property, ok := valid.idToProperty[int64(key.ID)]
				if !ok {
					blog.Errorf("[validCreateUnique] find [%s] property [%d] error %v", valid.objID, key.ID)
					return valid.errif.Errorf(common.CCErrTopoObjectPropertyNotFound, key.ID)
				}
				uniquekeys[property.PropertyID] = true
			default:
				blog.Errorf("[validCreateUnique] find [%s] property [%d] unique kind invalid [%d]", valid.objID, key.ID, key.Kind)
				return valid.errif.Errorf(common.CCErrTopoObjectUniqueKeyKindInvalid, key.Kind)
			}
		}

		cond := mongo.NewCondition()

		allEmpty := true
		for key := range uniquekeys {
			val, ok := instanceData[key]
			cond.Element(&mongo.Eq{Key: key, Val: val})
			if ok && !isEmpty(val) {
				allEmpty = false
			}
		}

		if allEmpty && !unique.MustCheck {
			continue
		}

		// only search data not in diable status
		cond.Element(&mongo.Neq{Key: common.BKDataStatusField, Val: common.DataStatusDisabled})
		if common.GetObjByType(valid.objID) == common.BKInnerObjIDObject {
			cond.Element(&mongo.Eq{Key: common.BKObjIDField, Val: valid.objID})
		}

		isExsit, bizID := instMedataData.Label.Get(common.BKAppIDField)
		if isExsit {
			_, metaCond := cond.Embed(metadata.BKMetadata)
			_, lableCond := metaCond.Embed(metadata.BKLabel)
			lableCond.Element(&mongo.Eq{Key: common.BKAppIDField, Val: bizID})
		}

		searchCond := metadata.QueryCondition{Condition: cond.ToMapStr()}
		result, err := instanceManager.SearchModelInstance(ctx, valid.objID, searchCond)
		if nil != err {
			blog.Errorf("[validCreateUnique] search [%s] inst error %v", valid.objID, err)
			return err
		}

		if 0 < result.Count {
			blog.Errorf("[validCreateUnique] duplicate data condition: %#v, unique keys: %#v, objID %s", cond.ToMapStr(), uniquekeys, valid.objID)
			return valid.errif.Error(common.CCErrCommDuplicateItem)
		}

	}

	return nil
}

// validUpdateUnique valid update unique
func (valid *validator) validUpdateUnique(ctx core.ContextParams, instanceData mapstr.MapStr, instMedataData metadata.Metadata, instID uint64, instanceManager *instanceManager) error {
	mapData, err := instanceManager.getInstDataByID(ctx, valid.objID, instID, instanceManager)
	if nil != err {
		blog.Errorf("[validUpdateUnique] search [%s] inst error %v", valid.objID, err)
		return err
	}

	// retrive isonly value
	for key, val := range instanceData {
		mapData[key] = val
	}

	uniqueAttr, err := valid.dependent.SearchUnique(ctx, valid.objID)
	if nil != err {
		blog.Errorf("[validUpdateUnique] search [%s] unique error %v", valid.objID, err)
		return err
	}

	if 0 >= len(uniqueAttr) {
		blog.Warnf("[validUpdateUnique] there're not unique constraint for %s, return", valid.objID)
		return nil
	}

	for _, unique := range uniqueAttr {
		// retrive unique value
		uniquekeys := map[string]bool{}
		for _, key := range unique.Keys {
			switch key.Kind {
			case metadata.UniqueKeyKindProperty:
				property, ok := valid.idToProperty[int64(key.ID)]
				if !ok {
					blog.Errorf("[validUpdateUnique] find [%s] property [%d] error %v", valid.objID, key.ID)
					return valid.errif.Errorf(common.CCErrTopoObjectPropertyNotFound, property.ID)
				}
				uniquekeys[property.PropertyID] = true
			default:
				blog.Errorf("[validUpdateUnique] find [%s] property [%d] unique kind invalid [%d]", valid.objID, key.ID, key.Kind)
				return valid.errif.Errorf(common.CCErrTopoObjectUniqueKeyKindInvalid, key.Kind)
			}
		}

		cond := mongo.NewCondition()
		allEmpty := true
		for key := range uniquekeys {
			val, ok := instanceData[key]
			cond.Element(&mongo.Eq{Key: key, Val: val})
			if ok && !isEmpty(val) {
				allEmpty = false
			}
		}

		if allEmpty && !unique.MustCheck {
			continue
		}

		// only search data not in diable status
		cond.Element(&mongo.Neq{Key: common.BKDataStatusField, Val: common.DataStatusDisabled})
		if common.GetObjByType(valid.objID) == common.BKInnerObjIDObject {
			cond.Element(&mongo.Eq{Key: common.BKObjIDField, Val: valid.objID})
		}
		cond.Element(&mongo.Neq{Key: common.GetInstIDField(valid.objID), Val: instID})
		isExsit, bizID := instMedataData.Label.Get(common.BKAppIDField)
		if isExsit {
			_, metaCond := cond.Embed(metadata.BKMetadata)
			_, lableCond := metaCond.Embed(metadata.BKLabel)
			lableCond.Element(&mongo.Eq{Key: common.BKAppIDField, Val: bizID})
		}

		searchCond := metadata.QueryCondition{Condition: cond.ToMapStr()}
		result, err := instanceManager.SearchModelInstance(ctx, valid.objID, searchCond)
		if nil != err {
			blog.Errorf("[validUpdateUnique] search [%s] inst error %v", valid.objID, err)
			return err
		}

		if 0 < result.Count {
			blog.Errorf("[validUpdateUnique] duplicate data condition: %#v, origin: %#v, unique keys: %v, objID: %s, instID %v count %d", cond.ToMapStr(), mapData, uniquekeys, valid.objID, instID, result.Count)
			return valid.errif.Error(common.CCErrCommDuplicateItem)
		}
	}
	return nil
}